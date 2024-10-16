package config

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"

	"google.golang.org/grpc"

	pb "github.com/orirawlings/gh-biome/internal/config/protobuf"

	"github.com/go-git/go-git/v5/plumbing/format/config"
)

type Config = config.Config

type Editor interface {
	Edit(context.Context, func(context.Context, *Config) (bool, error)) error
}

type editor struct {
	repoPath  string
	helperCmd string
}

func NewEditor(repoPath string, opts ...EditorOption) Editor {
	e := editor{
		repoPath:  repoPath,
		helperCmd: fmt.Sprintf("%s config-edit-helper", os.Args[0]),
	}
	for _, opt := range opts {
		opt(&e)
	}
	return &e
}

func (e *editor) Edit(ctx context.Context, do func(context.Context, *Config) (bool, error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lis, err := e.listener(ctx)
	if err != nil {
		return err
	}
	defer lis.Close()

	// start server
	s := newEditorServer()
	gs := grpc.NewServer()
	pb.RegisterEditorServer(gs, s)
	defer gs.GracefulStop()
	go gs.Serve(lis)

	// start git config editor
	cmd := exec.CommandContext(ctx, "git", "-C", e.repoPath, "config", "edit", "--local")
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_EDITOR=%s %s", e.helperCmd, lis.Addr()))
	var out bytes.Buffer
	cmd.Stderr = &out
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not %q: %w", cmd, err)
	}
	cmdErr := make(chan error)
	go func() {
		defer close(cmdErr)
		if err := cmd.Wait(); err != nil {
			select {
			case cmdErr <- fmt.Errorf("could not %q: %w", cmd, err):
			case <-ctx.Done():
			}
		}
	}()

	var path string
	select {
	case err := <-cmdErr:
		if err != nil {
			err = fmt.Errorf("%q ended unexpectedly early: %w\n%s", cmd, err, out.String())
		} else {
			err = fmt.Errorf("%q ended unexpectedly early:\n%s", cmd, out.String())
		}
		return err
	case path = <-s.Path():
	case <-ctx.Done():
		return ctx.Err()
	}

	// parse the config file
	cfg, err := e.load(path)
	if err != nil {
		err = fmt.Errorf("could not load config file: %w", err)
		defer s.Done(ctx, err)
		return err
	}

	save, err := do(ctx, cfg)
	if err != nil {
		err = fmt.Errorf("editor callback failed: %w", err)
		defer s.Done(ctx, err)
		return err
	}

	if save {
		if err := e.save(path, cfg); err != nil {
			err = fmt.Errorf("could not save config file: %w", err)
			defer s.Done(ctx, err)
			return err
		}
	}

	defer s.Done(ctx, nil)
	return nil
}

func (e *editor) listener(ctx context.Context) (net.Listener, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temp file")
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("could not close temp file")
	}
	if err := os.Remove(f.Name()); err != nil {
		return nil, fmt.Errorf("could not remove temp file")
	}
	return (&net.ListenConfig{}).Listen(ctx, "unix", f.Name())
}

func (e *editor) load(path string) (*Config, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cfg := config.New()
	defer configFile.Close()
	return cfg, config.NewDecoder(configFile).Decode(cfg)
}

func (e *editor) save(path string, cfg *Config) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()
	return config.NewEncoder(w).Encode(cfg)
}

type EditorOption func(*editor)

// HelperCommand overrides the command forked by 'git config edit' that will
// callback to the editor server. The command will be passed two arguments, the
// unix socket to call the server and the temporary config file to edit
// provided by 'git config edit'.
//
// Generally, this option will only be used during tests.
func HelperCommand(cmd string) EditorOption {
	return func(e *editor) {
		e.helperCmd = cmd
	}
}

type editorServer struct {
	pb.UnimplementedEditorServer
	path  chan string
	errCh chan error
}

func newEditorServer() *editorServer {
	return &editorServer{
		path:  make(chan string),
		errCh: make(chan error),
	}
}

func (e *editorServer) Edit(ctx context.Context, req *pb.EditRequest) (*pb.Empty, error) {
	select {
	case e.path <- req.Path:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	close(e.path)

	select {
	case err := <-e.errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *editorServer) Path() chan string {
	return e.path
}

func (e *editorServer) Done(ctx context.Context, err error) error {
	defer close(e.errCh)
	if err != nil {
		select {
		case e.errCh <- err:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
