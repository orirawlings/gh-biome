package biome

import (
	"testing"
)

var (
	github_com_cli = Owner{
		host: "github.com",
		name: "cli",
	}

	github_com_git = Owner{
		host: "github.com",
		name: "git",
	}

	github_com_kubernetes = Owner{
		host: "github.com",
		name: "kubernetes",
	}

	github_com_orirawlings = Owner{
		host: "github.com",
		name: "orirawlings",
	}

	my_github_biz_foobar = Owner{
		host: "my.github.biz",
		name: "foobar",
	}

	owners = []Owner{
		github_com_cli,
		github_com_git,
		github_com_kubernetes,
		github_com_orirawlings,
		my_github_biz_foobar,
	}

	ownerIds = map[string]string{
		github_com_cli.String():         "MDEyOk9yZ2FuaXphdGlvbjU5NzA0NzEx",
		github_com_git.String():         "MDEyOk9yZ2FuaXphdGlvbjE4MTMz",
		github_com_kubernetes.String():  "MDEyOk9yZ2FuaXphdGlvbjEzNjI5NDA4",
		github_com_orirawlings.String(): "MDQ6VXNlcjU3MjEz",
		my_github_biz_foobar.String():   "foobar",
	}
)

func TestParseOwner(t *testing.T) {
	type run struct {
		ownerRef string
		expected Owner
		invalid  bool
	}
	runs := []run{
		{
			ownerRef: "orirawlings",
			expected: github_com_orirawlings,
		},
		{
			ownerRef: "https://github.com/orirawlings",
			expected: github_com_orirawlings,
		},
		{
			ownerRef: "GitHub.com/orirawlings",
			expected: github_com_orirawlings,
		},
		{
			ownerRef: "https://foobar",
			invalid:  true,
		},
		{
			ownerRef: "https://",
			invalid:  true,
		},
		{
			ownerRef: "foo/bar/baz",
			invalid:  true,
		},
		{
			ownerRef: "",
			invalid:  true,
		},
	}
	for _, o := range owners {
		runs = append(runs, run{
			ownerRef: o.String(),
			expected: o,
		})
	}
	for _, r := range runs {
		t.Run(string(r.ownerRef), func(t *testing.T) {
			o, err := ParseOwner(r.ownerRef)
			if r.invalid {
				if err == nil {
					t.Error("expected parse error, but was nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected parse error: %v", err)
				}
				if o != r.expected {
					t.Errorf("unexpected: wanted %v, was %v", r.expected, o)
				}
			}
		})
	}
}

func TestOwner_RemoteGroup(t *testing.T) {
	for _, run := range []struct {
		owner    Owner
		expected string
	}{
		{
			owner:    github_com_cli,
			expected: "g-bfededc94e5a455909e7c6f37745542a029502b5",
		},
		{
			owner:    github_com_git,
			expected: "g-a4a59a714b275266370b0f37c8205b46f6d7acdc",
		},
		{
			owner:    github_com_kubernetes,
			expected: "g-e6659b2160a690755a425e155b330c634ab6dd8d",
		},
		{
			owner:    github_com_orirawlings,
			expected: "g-09d78f0998d14107e2e01273c44dba15b5ad70d0",
		},
		{
			owner:    my_github_biz_foobar,
			expected: "g-c29aef5516494c069810c89978e17e0acf799a49",
		},
	} {
		t.Run(run.owner.String(), func(t *testing.T) {
			g := run.owner.RemoteGroup()
			if g != run.expected {
				t.Errorf("wanted %q, was %q", run.expected, g)
			}
		})
	}
}
