package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SnapshotOptions configures sandbox snapshot and rollback operations.
type SnapshotOptions struct {
	BoiteBin string
	Runner   Runner
}

// DefaultSnapshotOptions returns standard snapshot options.
func DefaultSnapshotOptions() SnapshotOptions {
	return SnapshotOptions{
		BoiteBin: "boite",
		Runner:   NewDefaultRunner(),
	}
}

func resolveBoiteBin(bin string) string {
	if bin != "" {
		return bin
	}
	return "boite"
}

func resolveRunner(runner Runner) Runner {
	if runner != nil {
		return runner
	}
	return NewDefaultRunner()
}

// TakeSnapshot creates a snapshot of the stopped or active sandbox overlay disk.
func TakeSnapshot(ctx context.Context, name string, tag string, opts SnapshotOptions) error {
	if name == "" {
		return errors.New("instance name is required")
	}
	if tag == "" {
		return errors.New("snapshot tag is required")
	}
	bin := resolveBoiteBin(opts.BoiteBin)
	runner := resolveRunner(opts.Runner)
	out, err := runner.Run(ctx, bin, "snapshot", name, tag)
	if err != nil {
		return fmt.Errorf("failed to take snapshot %s for %s: %w (%s)", tag, name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ListSnapshots returns the list of available snapshot tags for an instance.
func ListSnapshots(ctx context.Context, name string, opts SnapshotOptions) ([]string, error) {
	if name == "" {
		return nil, errors.New("instance name is required")
	}
	bin := resolveBoiteBin(opts.BoiteBin)
	runner := resolveRunner(opts.Runner)
	out, err := runner.Run(ctx, bin, "snapshots", name)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	var tags []string
	for line := range strings.SplitSeq(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags, nil
}

// RollbackSnapshot restores an instance overlay disk to the specified snapshot tag.
func RollbackSnapshot(ctx context.Context, name string, tag string, opts SnapshotOptions) error {
	if name == "" {
		return errors.New("instance name is required")
	}
	if tag == "" {
		return errors.New("snapshot tag is required")
	}
	bin := resolveBoiteBin(opts.BoiteBin)
	runner := resolveRunner(opts.Runner)
	out, err := runner.Run(ctx, bin, "rollback", name, tag)
	if err != nil {
		return fmt.Errorf("failed to rollback %s to %s: %w (%s)", name, tag, err, strings.TrimSpace(string(out)))
	}
	return nil
}
