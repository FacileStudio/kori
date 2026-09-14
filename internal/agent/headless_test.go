package agent

import (
	"os"
	"strings"
	"testing"
)

func TestStripPrintFlagExtractsPrompt(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		want     string
		wantArgs string
	}{
		{"-print with arg", []string{"kori", "-print", "hello world"}, "hello world", "kori"},
		{"-print=equals", []string{"kori", "-print=hello"}, "hello", "kori"},
		{"-print alone (stdin)", []string{"kori", "-print"}, "", "kori"},
		{"no -print", []string{"kori", "-model", "abc"}, "", "kori -model abc"},
		{"-print after other flags", []string{"kori", "-root", ".", "-print", "hello"}, "hello", "kori -root ."},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			saved := os.Args
			os.Args = tt.args
			got := stripPrintFlag()
			gotArgs := strings.Join(os.Args, " ")
			os.Args = saved

			if got != tt.want {
				t.Errorf("stripPrintFlag() = %q, want %q", got, tt.want)
			}
			if gotArgs != tt.wantArgs {
				t.Errorf("after strip, os.Args = %q, want %q", gotArgs, tt.wantArgs)
			}
		})
	}
}
