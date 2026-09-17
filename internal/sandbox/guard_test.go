package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildGuardProbeCommand(t *testing.T) {
	cmd1 := BuildGuardProbeCommand("")
	if !strings.Contains(cmd1, "whoami") || !strings.Contains(cmd1, "VM_HARNESS=") {
		t.Fatalf("unexpected probe command: %s", cmd1)
	}
	cmd2 := BuildGuardProbeCommand("/workspace")
	if !strings.Contains(cmd2, "WORKDIR_OK") || !strings.Contains(cmd2, "/workspace") {
		t.Fatalf("unexpected probe command with workdir: %s", cmd2)
	}
}

func TestParseGuardOutput(t *testing.T) {
	output := "boite\n/home/boite\nVM_HARNESS=1726000000\nWORKDIR_OK\n"
	res, err := ParseGuardOutput(output)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if res.User != "boite" || res.Workdir != "/home/boite" || res.Harness != "1726000000" {
		t.Fatalf("unexpected guard result: %+v", res)
	}
	if _, err := ParseGuardOutput("single_line"); err == nil {
		t.Fatal("expected error for insufficient lines")
	}
}

func TestVerifyIsolation(t *testing.T) {
	opts := GuardOptions{ExpectedUser: "boite", CheckHarness: true, ExpectedWorkdir: "/workspace"}
	res := &GuardResult{User: "root", Workdir: "/root", Harness: "123"}
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err == nil {
		t.Fatal("expected error for root user")
	}
	res.User = "other"
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err == nil {
		t.Fatal("expected error for user mismatch")
	}
	res.User = "boite"
	res.Harness = ""
	if err := VerifyIsolation(res, "WORKDIR_OK", opts); err == nil {
		t.Fatal("expected error for missing harness")
	}
	res.Harness = "123"
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
			return []byte("boite\n/home/boite\nVM_HARNESS=12345\n"), nil
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
