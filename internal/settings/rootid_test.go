package settings

import "testing"

func TestTargetRootLabelsTheTarget(t *testing.T) {
	if got := TargetRoot("pingu"); got != "target:pingu" {
		t.Errorf("TargetRoot(\"pingu\") = %q, want \"target:pingu\"", got)
	}
	if got := TargetRoot(""); got != "" {
		t.Errorf("TargetRoot(\"\") = %q, want an empty root, not a target label", got)
	}
}

func TestIsTargetRoot(t *testing.T) {
	for root, want := range map[string]bool{
		"target:pingu":      true,
		"target:":           true,
		"/home/yann/Code":   false,
		".":                 false,
		"":                  false,
		"/tmp/target:weird": false,
		"target:my-vm:2226": true,
	} {
		if got := IsTargetRoot(root); got != want {
			t.Errorf("IsTargetRoot(%q) = %v, want %v", root, got, want)
		}
	}
}

func TestTargetRootName(t *testing.T) {
	if got := TargetRootName("target:pingu"); got != "pingu" {
		t.Errorf("TargetRootName(\"target:pingu\") = %q, want \"pingu\"", got)
	}
	for _, root := range []string{"", ".", "/srv/app"} {
		if got := TargetRootName(root); got != "" {
			t.Errorf("TargetRootName(%q) = %q, want \"\" for a plain path", root, got)
		}
	}
}
