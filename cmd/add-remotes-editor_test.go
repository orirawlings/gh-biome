package cmd

import (
	"testing"
)

func TestRemoteGroup(t *testing.T) {
	r := remote{
		Name: "github.com/orirawlings/gh-ubergit", // sha1 = bbf0efd9345e9f7d238c1064d74a7166f9103bce
	}
	group := r.Group()
	if group != "ubergit-bb" {
		t.Errorf("was: %s", group)
	}
}
