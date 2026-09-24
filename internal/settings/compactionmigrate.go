package settings

import "fmt"

// This file is the migration surface: the keys this release renamed or removed,
// and the refusal that names where each went. It lives apart from compaction.go
// because a migration is a different question from a validation — one is about a
// config written against an older release, the other about one written against
// this one.

// removedCompactionEnv refuses a ratio variable this release renamed or removed.
//
// The file layer gets this for free: KnownFields(true) rejects `mid_ratio` by
// name. The environment layer cannot, and that leniency is deliberate — a value
// it cannot read is treated as unmentioned, so a misspelt boolean falls through
// to the layer below rather than silently pinning false. The cost of it is that a
// stale variable is simply ignored, and for a ratio that means the session quietly
// runs on the default instead of the figure its owner tuned. There is no other
// signal at all in that case, and the fix is a one-line rename, so it is worth
// the hard refusal.
//
// A variable set to the empty string reads as unmentioned, the way every other
// value in this layer does.
func removedCompactionEnv() error {
	for _, gone := range []struct{ name, hint string }{
		{"COMPACTION_MID_RATIO", "it is now " + EnvPrefix + "COMPACTION_SMART_RATIO"},
		{"COMPACTION_HARD_RATIO", "it went with the third tier, whose forced fold is now derived rather than configured"},
	} {
		if v, ok := lookup(EnvPrefix + gone.name); !ok || v == "" {
			continue
		}
		return &ParseError{Path: EnvPrefix + gone.name, Err: fmt.Errorf(
			"this variable is no longer read — %s", gone.hint)}
	}
	return nil
}
