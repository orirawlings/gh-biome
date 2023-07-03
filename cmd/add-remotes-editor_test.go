package cmd

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/config"
)

func TestUpdateConfig(t *testing.T) {
	cfg := config.New()
	remotes := byName([]remote{
		{
			Name:     "github.com/foo/bar",
			FetchURL: "git@github.com:foo/bar.git",
		},
		{
			Name:     "github.com/foo/archived",
			FetchURL: "git@github.com:foo/archived.git",
			Archived: true,
		},
		{
			Name:     "github.com/foo/disabled",
			FetchURL: "git@github.com:foo/disabled.git",
			Disabled: true,
		},
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

func byName(remotes []remote) map[string]remote {
	result := make(map[string]remote)
	for _, r := range remotes {
		result[r.Name] = r
	}
	return result
}
