package main

import (
	"errors"
	"os"
	"strings"

	"github.com/FacileStudio/kori/cmd"
	"github.com/FacileStudio/kori/internal/agent"
)

var version = "v0.66.0"

func main() {
	if err := cmd.Execute(version); err != nil {
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
