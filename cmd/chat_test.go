package cmd

import (
	"bytes"
	"testing"
)

func TestChatHelpCommand(t *testing.T) {
	cmd := newRootCmd("v0.77.0")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"chat", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error on chat help, got: %v", err)
	}
	for _, sub := range []string{"channels", "install", "uninstall", "verify"} {
		if !bytes.Contains(buf.Bytes(), []byte(sub)) {
			t.Errorf("expected %s subcommand in chat help, got: %s", sub, buf.String())
		}
	}
}

func TestChatVerifyHelpCommand(t *testing.T) {
	cmd := newRootCmd("v0.77.0")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"chat", "verify", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error on chat verify help, got: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("--recovery-key")) {
		t.Errorf("expected --recovery-key flag in chat verify help, got: %s", buf.String())
	}
}
