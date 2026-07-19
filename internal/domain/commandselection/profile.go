// Package commandselection defines the persisted, exact command-path allowlist
// used to derive cwk's active attention surface.
package commandselection

import (
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	// MaxEnabledCommands bounds the persisted selection independently of the
	// number of commands in any particular release.
	MaxEnabledCommands = 512
	// MaxCommandPathBytes keeps one syntactically valid but hostile path from
	// consuming the profile's entire storage budget.
	MaxCommandPathBytes = 512
)

// Profile is an ordered allowlist of canonical exact command paths. Order is
// preserved so the CLI can persist catalog order deterministically. The slice
// remains private and every boundary copies it.
type Profile struct {
	enabled []string
}

// New validates and snapshots an ordered command selection. Unknown command
// paths are allowed: they are retained as stale upgrade evidence and resolved
// against the complete catalog only by the CLI layer.
func New(enabled []string) (Profile, error) {
	if len(enabled) > MaxEnabledCommands {
		return Profile{}, fmt.Errorf("enabled command count exceeds %d", MaxEnabledCommands)
	}

	copyOfEnabled := make([]string, len(enabled))
	copy(copyOfEnabled, enabled)
	seen := make(map[string]struct{}, len(copyOfEnabled))
	for index, path := range copyOfEnabled {
		if len(path) > MaxCommandPathBytes {
			return Profile{}, fmt.Errorf("enabled command %d exceeds %d bytes", index, MaxCommandPathBytes)
		}
		if err := operation.ValidateCommandPath(path); err != nil {
			return Profile{}, fmt.Errorf("enabled command %d: %w", index, err)
		}
		if _, duplicate := seen[path]; duplicate {
			return Profile{}, fmt.Errorf("enabled command %d duplicates an earlier path", index)
		}
		seen[path] = struct{}{}
	}
	return Profile{enabled: copyOfEnabled}, nil
}

// EnabledCommands returns an independent copy of the ordered allowlist.
func (p Profile) EnabledCommands() []string {
	commands := make([]string, len(p.enabled))
	copy(commands, p.enabled)
	return commands
}

// Contains reports whether the exact canonical path is enabled.
func (p Profile) Contains(path string) bool {
	for _, enabled := range p.enabled {
		if enabled == path {
			return true
		}
	}
	return false
}

// Validate rechecks the value's invariants. It protects application boundaries
// even if a zero value or a value copied through future domain code is used.
func (p Profile) Validate() error {
	_, err := New(p.enabled)
	return err
}
