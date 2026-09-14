package settings

import (
	"testing"
)

// On by default: delegation is enabled out of the box.
func TestParallelAgentsDefaultOn(t *testing.T) {
	written(t, "")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if !*config.ParallelAgents {
		t.Error("parallel_agents default off")
	}
}

// The whole precedence chain has to carry the toggle, or the layer that
// turns it on is not the layer that decides.
func TestParallelAgentsFollowThePrecedenceChain(t *testing.T) {
	written(t, "tools:\n  parallel_agents: false")
	t.Setenv("KORI_PARALLEL_AGENTS", "true")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if !*config.ParallelAgents {
		t.Error("the environment did not beat the file")
	}
}

// The settings layer can only resolve the yaml into the config; mounting the
// tool is agent wiring, asserted in the agent package's delegate_test.go. What
// the layer owes that mount is that parallel_agents: true reaches the resolved
// config.
func TestParallelAgentsTrueInTheFileTurnsTheMountOn(t *testing.T) {
	written(t, "tools:\n  parallel_agents: true")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if !*config.ParallelAgents {
		t.Error("parallel_agents: true in the file left the mount off")
	}
}
