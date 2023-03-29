//go:build tools
// +build tools

package tools

// Tools we use during development.
import (
	_ "github.com/mgechev/revive"
	_ "go.abhg.dev/requiredfield/cmd/requiredfield"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
