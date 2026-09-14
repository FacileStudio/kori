package settings

import (
	"os"
	"testing"
)

func TestLegacyNacelleEnvStillResolves(t *testing.T) {
	t.Setenv("NACELLE_BACKEND", "legacy")
	t.Setenv("KORI_BACKEND", "current")
	if got := envGet("BACKEND"); got != "current" {
		t.Fatalf("KORI_ must win over NACELLE_: got %q", got)
	}

	t.Setenv("KORI_BACKEND", "")
	if err := os.Unsetenv("KORI_BACKEND"); err != nil {
		t.Fatal(err)
	}
	if got := providerEnv().Backend; got != "legacy" {
		t.Fatalf("NACELLE_BACKEND must still resolve: got %q", got)
	}
}
