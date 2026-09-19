package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildGuardProbeCommand(t *testing.T) {
	cmd1 := BuildGuardProbeCommand("")
	if !strings.Contains(cmd1, "whoami") || !strings.Contains(cmd1, "pwd") {
		t.Fatalf("unexpected probe command: %s", cmd1)
	}
	cmd2 := BuildGuardProbeCommand("/workspace")
	if !strings.Contains(cmd2, "WORKDIR_OK") || !strings.Contains(cmd2, "/workspace") {
		t.Fatalf("unexpected probe command with workdir: %s", cmd2)
	}
}

func TestParseGuardOutput(t *testing.T) {
	output := "boite\n/home/boite\nWORKDIR_OK\n"
	res, err := ParseGuardOutput(output)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if res.User != "boite" || res.Workdir != "/home/boite" {
		t.Fatalf("unexpected guard result: %+v", res)
	}
	if _, err := ParseGuardOutput("single_line"); err == nil {
		t.Fatal("expected error for insufficient lines")
	}
}

func TestVerifyIsolation(t *testing.T) {
	opts := GuardOptions{ExpectedUser: "boite", ExpectedWorkdir: "/workspace"}
	res := &GuardResult{User: "root", Workdir: "/root"}
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err == nil {
		t.Fatal("expected error when root does not match the expected user")
	}
	if err := VerifyIsolation(res, "WORKDIR_OK", GuardOptions{ExpectedUser: "root"}); err != nil {
		t.Fatalf("expected root to be allowed when the target asks for it: %v", err)
	}
	if err := VerifyIsolation(res, "WORKDIR_OK", GuardOptions{}); err != nil {
		t.Fatalf("expected root to be allowed with no expectation set: %v", err)
	}
	res.User = "other"
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err == nil {
		t.Fatal("expected error for user mismatch")
	}
	res.User = "boite"
	if err := VerifyIsolation(res, "WORKDIR_MISSING", opts); err == nil {
		t.Fatal("expected error for missing workdir")
	}
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreflightCheck(t *testing.T) {
	target := &Target{Name: "pingu", Backend: "boite", Port: 2226, KeyPath: "/key", Status: "stopped"}
	opts := DefaultGuardOptions()
	if _, err := PreflightCheck(context.Background(), target, opts); err == nil {
		t.Fatal("expected error for stopped VM")
	}
	target.Status = "running"
	opts.Runner = &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("boite\n/home/boite\n"), nil
		},
	}
	res, err := PreflightCheck(context.Background(), target, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.User != "boite" {
		t.Fatalf("expected boite user, got %s", res.User)
	}
}

func TestPreflightCheckFailures(t *testing.T) {
	if _, err := PreflightCheck(context.Background(), nil, DefaultGuardOptions()); err == nil {
		t.Fatal("expected error for nil instance")
	}
	target := &Target{Name: "pingu", Backend: "boite", Port: 2226, KeyPath: "/key", Status: "running"}
	opts := GuardOptions{
		Runner: &mockRunner{
			runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, errors.New("connection refused")
			},
		},
	}
	if _, err := PreflightCheck(context.Background(), target, opts); err == nil {
		t.Fatal("expected error when ssh runner fails")
	}
}
