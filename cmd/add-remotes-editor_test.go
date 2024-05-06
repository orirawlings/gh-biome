package cmd

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/config"
)

func init() {
	addRemotesEditorCmd.SetContext(context.Background())
	pushInContext(addRemotesEditorCmd)
}

func TestUpdateConfig(t *testing.T) {
	ctx := addRemotesEditorCmd.Context()

	cfg := config.New()
	remotes := byName([]remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
		dotPrefixRemote,
	})
	updateConfig(ctx, cfg, remotes)
	for _, r := range remotes {
		if r.Disabled {
			if cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("should not have sub-section: %q", "remote."+r.Name)
			}
			continue
		}
		if r.Supported() {
			if !cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("missing sub-section: %q", "remote."+r.Name)
			}
		} else {
			if cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("unexpected sub-section, expected this remote to be skipped during configuration: %q", "remote."+r.Name)
			}
		}
	}
}

func TestAddRemotesEditorCmd_Execute(t *testing.T) {
	remotes := byName([]remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
		dotPrefixRemote,
	})

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if err := remotes.save(f); err != nil {
		t.Fatalf("could not save remotes data for test: %v", err)
	}
	f.Close()

	gitconfig, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gitconfig.Name())
	gitconfig.Close()

	rootCmd.SetArgs([]string{
		"add-remotes-editor",
		f.Name(),
		gitconfig.Name(),
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}

	configFile, err := os.Open(gitconfig.Name())
	if err != nil {
		t.Fatalf("could not open git config file for assertion: %v", err)
	}
	defer configFile.Close()
	cfg := config.New()
	config.NewDecoder(configFile).Decode(cfg)

	for _, r := range remotes {
		if r.Disabled {
			if cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("should not have sub-section: %q", "remote."+r.Name)
			}
			continue
		}
		if r.Supported() {
			if !cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("missing sub-section: %q", "remote."+r.Name)
			}
		} else {
			if cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("unexpected sub-section, expected this remote to be skipped during configuration: %q", "remote."+r.Name)
			}
		}
	}

	// include config file in temporary git repo and ensure remote configs are valid
	repo, _ := tempGitRepo(t)
	cmd := exec.Command("git", "-C", repo, "config", "include.path", gitconfig.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("could not include test config in temp repo: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "-C", repo, "remote", "show")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("could not list remotes in temp repo: %v\n%s", err, string(out))
	}
	var actualRemotes []string
	for _, r := range strings.Split(string(out), "\n") {
		r := strings.TrimSpace(r)
		if len(r) > 0 {
			actualRemotes = append(actualRemotes, r)
		}
	}
	expectedRemotes := byName([]remote{
		barRemote,
		archivedRemote,
		headlessRemote,
	})
	if len(expectedRemotes) != len(actualRemotes) {
		t.Errorf("expected %d remotes to be configured on temp git repo, but was %d", len(expectedRemotes), len(actualRemotes))
	}
	for _, r := range actualRemotes {
		if _, ok := expectedRemotes[r]; !ok {
			t.Errorf("found unexpected remote configured on temp git repo: %s", r)
		}
	}
}
