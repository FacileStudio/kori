package sandbox

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultSnapshotOptions(t *testing.T) {
	opts := DefaultSnapshotOptions()
	if opts.BoiteBin != "boite" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestTakeSnapshot(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name != "boite" || len(args) < 3 || args[0] != "snapshot" {
				return nil, errors.New("invalid args")
			}
			return []byte("snapshot created"), nil
		},
	}
	opts := SnapshotOptions{Runner: runner}
	if err := TakeSnapshot(context.Background(), "pingu", "v1", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTakeSnapshotValidation(t *testing.T) {
	opts := DefaultSnapshotOptions()
	if err := TakeSnapshot(context.Background(), "", "v1", opts); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := TakeSnapshot(context.Background(), "pingu", "", opts); err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestListSnapshots(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("tag1\ntag2\n"), nil
		},
	}
	opts := SnapshotOptions{Runner: runner}
	tags, err := ListSnapshots(context.Background(), "pingu", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "tag1" || tags[1] != "tag2" {
		t.Fatalf("unexpected tags: %v", tags)
	}
	if _, err := ListSnapshots(context.Background(), "", opts); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRollbackSnapshot(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name != "boite" || len(args) < 3 || args[0] != "rollback" {
				return nil, errors.New("invalid args")
			}
			return []byte("rolled back"), nil
		},
	}
	opts := SnapshotOptions{Runner: runner}
	if err := RollbackSnapshot(context.Background(), "pingu", "tag1", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := RollbackSnapshot(context.Background(), "", "tag1", opts); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := RollbackSnapshot(context.Background(), "pingu", "", opts); err == nil {
		t.Fatal("expected error for empty tag")
	}
}
