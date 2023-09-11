package cmd

import (
	"os"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/config"
)

func TestUpdateConfig(t *testing.T) {
	cfg := config.New()
	remotes := byName([]remote{
		barRemote,
		archivedRemote,
		disabledRemote,
		lockedRemote,
		headlessRemote,
	})
	updateConfig(cfg, remotes)
	for _, r := range remotes {
		if r.Disabled {
			if cfg.Section("remote").HasSubsection(r.Name) {
				t.Errorf("should not have sub-section: %q", "remote."+r.Name)
			}
			continue
		}
		if !cfg.Section("remote").HasSubsection(r.Name) {
			t.Errorf("missing sub-section: %q", "remote."+r.Name)
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
		if !cfg.Section("remote").HasSubsection(r.Name) {
			t.Errorf("missing sub-section: %q", "remote."+r.Name)
		}
	}
}
