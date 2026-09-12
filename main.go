package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/FacileStudio/nacelle-tui/internal/agent"
)

var version = "v0.53.2"

func main() {
	if err := agent.Run(version); err != nil {
		fmt.Fprintln(os.Stderr, "nacelle:", unprefixed(err))
		var usage *agent.UsageError
		if errors.As(err, &usage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func unprefixed(err error) string {
	return strings.TrimPrefix(err.Error(), "nacelle: ")
}
