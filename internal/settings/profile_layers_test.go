package settings

import "testing"

// A profile is a layer of its own, so it beats the file that named it — and
// the environment and the flags still beat the profile, field by field, which
// is what lets one profile serve every checkout without overriding what is
// specific to this machine.
func TestAProfileBeatsTheFileAndLosesToTheEnvironment(t *testing.T) {
	setupProfileEnvWith(t, "slow.yml",
		"name: slow\nlimits:\n  max_iterations: 12\n",
		"profile: slow\nlimits:\n  max_iterations: 3\n")

	cfg, err := Settings("", Config{})
	if err != nil {
		t.Fatalf("Settings failed: %v", err)
	}
	if *cfg.MaxIterations != 12 {
		t.Errorf("max iterations = %d, want the profile's 12 over the file's 3", *cfg.MaxIterations)
	}
	if cfg.Profile != "slow" {
		t.Errorf("profile = %q, want the selected profile recorded", cfg.Profile)
	}

	t.Setenv("KORI_MAX_ITERATIONS", "8")
	if cfg, err := Settings("", Config{}); err != nil {
		t.Fatalf("Settings failed: %v", err)
	} else if *cfg.MaxIterations != 8 {
		t.Errorf("max iterations = %d, want the environment's 8 over the profile's 12", *cfg.MaxIterations)
	}

	flag := 5
	if cfg, err := Settings("", Config{Limits: Limits{MaxIterations: &flag}}); err != nil {
		t.Fatalf("Settings failed: %v", err)
	} else if *cfg.MaxIterations != 5 {
		t.Errorf("max iterations = %d, want the flag's 5 over everything", *cfg.MaxIterations)
	}
}

// The same file has to give the same answer however the profile was selected:
// the file's own profile: key, KORI_PROFILE, and -profile all resolve to the
// same layer, above ~/.kori.yml.
func TestEveryLayerThatNamesAProfileResolvesItTheSameWay(t *testing.T) {
	slow := "name: slow\nlimits:\n  max_iterations: 12\n"
	cases := []struct {
		name  string
		file  string
		env   string
		flags Config
	}{
		{"the file names it", "profile: slow\nlimits:\n  max_iterations: 3\n", "", Config{}},
		{"KORI_PROFILE names it", "limits:\n  max_iterations: 3\n", "slow", Config{}},
		{"-profile names it", "limits:\n  max_iterations: 3\n", "", Config{Profile: "slow"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupProfileEnvWith(t, "slow.yml", slow, c.file)
			if c.env != "" {
				t.Setenv("KORI_PROFILE", c.env)
			}

			cfg, err := Settings("", c.flags)
			if err != nil {
				t.Fatalf("Settings failed: %v", err)
			}
			if *cfg.MaxIterations != 12 {
				t.Errorf("max iterations = %d, want the profile's 12 over the file's 3", *cfg.MaxIterations)
			}
		})
	}
}

// -no-config drops ~/.kori.yml and nothing else, so a profile the flag or the
// environment names still applies — and the file's competing limit, which this
// run never reads, does not come back with it.
func TestNoConfigStillAppliesANamedProfile(t *testing.T) {
	setupProfileEnvWith(t, "slow.yml",
		"name: slow\nlimits:\n  max_iterations: 12\n",
		"limits:\n  max_iterations: 3\n")
	off := true
	t.Setenv("KORI_PROFILE", "slow")

	cfg, err := Settings("", Config{NoConfig: &off})
	if err != nil {
		t.Fatalf("Settings failed: %v", err)
	}
	if *cfg.MaxIterations != 12 {
		t.Errorf("max iterations = %d, want the profile's 12 with the file skipped", *cfg.MaxIterations)
	}

	cfg, err = Settings("", Config{NoConfig: &off, Profile: "slow"})
	if err != nil {
		t.Fatalf("Settings failed: %v", err)
	}
	if *cfg.MaxIterations != 12 {
		t.Errorf("max iterations = %d, want the profile's 12 from -profile", *cfg.MaxIterations)
	}
}
