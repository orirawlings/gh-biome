package biome

import (
	"testing"
)

func TestParseOwner(t *testing.T) {
	for _, run := range []struct {
		ownerRef string
		expected Owner
		invalid  bool
	}{
		{
			ownerRef: "orirawlings",
			expected: Owner{
				host: "github.com",
				name: "orirawlings",
			},
		},
		{
			ownerRef: "github.com/orirawlings",
			expected: Owner{
				host: "github.com",
				name: "orirawlings",
			},
		},
		{
			ownerRef: "https://github.com/orirawlings",
			expected: Owner{
				host: "github.com",
				name: "orirawlings",
			},
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
	} {
		t.Run(string(run.ownerRef), func(t *testing.T) {
			o, err := ParseOwner(run.ownerRef)
			if run.invalid {
				if err == nil {
					t.Error("expected parse error, but was nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected parse error: %v", err)
				}
				if o != run.expected {
					t.Errorf("unexpected: wanted %v, was %v", run.expected, o)
				}
			}
		})
	}
}
