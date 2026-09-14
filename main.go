package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/FacileStudio/kori/internal/agent"
)

var version = "v0.57.0"

func main() {
	if err := agent.Run(version); err != nil {
		fmt.Fprintln(os.Stderr, "kori:", unprefixed(err))
		var usage *agent.UsageError
		if errors.As(err, &usage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func unprefixed(err error) string {
	return strings.TrimPrefix(err.Error(), "kori: ")
}
