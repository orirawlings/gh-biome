package cmd

import (
	"os"
	"testing"

	testutil "github.com/orirawlings/gh-biome/internal/util/testing"
)

var (
	biomeBuildPath string
)

func TestMain(m *testing.M) {
	var err error
	biomeBuildPath, err = testutil.BiomeBuild()
	if err != nil {
		panic(err.Error())
	}
	defer os.Remove(biomeBuildPath)
	m.Run()
}
