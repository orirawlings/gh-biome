package command

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrintln(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	Println(cmd, "Hello", "World")
	expected := "Hello World\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}
