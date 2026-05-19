package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Profile is an isolated agent configuration directory definition.
//
// Description is a free-form, multi-line hint that helps AI agents (or the user)
// pick the right profile based on the user's prompt. It is *not* used for any
// runtime logic — only as discoverability metadata surfaced by `aipim list`
// and `aipim get`.
type Profile struct {
	Name        string `json:"name"`
	Alias       string `json:"alias,omitempty"`
	Path        string `json:"path"`
	AgentName   string `json:"agent"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// reservedIdentifiers are profile names/aliases that would collide with built-in
// cobra subcommands and therefore can never be reached via `aipim <identifier>`.
var reservedIdentifiers = map[string]bool{
	"launch":     true,
	"create":     true,
	"help":       true,
	"completion": true,
	"list":       true,
	"ls":         true,
	"get":        true,
	"show":       true,
	"edit":       true,
	"delete":     true,
	"agent":      true,
	"version":    true,
}

// ValidateName validates a proposed profile name against the existing profiles.
func ValidateName(name string, existing []Profile) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("profile name cannot be empty")
	}
	if err := checkIdentifierShape(trimmed, "profile name"); err != nil {
		return err
	}

	for _, candidate := range existing {
		if strings.EqualFold(candidate.Name, trimmed) {
			return fmt.Errorf("profile %q already exists", trimmed)
		}
		if candidate.Alias != "" && strings.EqualFold(strings.TrimSpace(candidate.Alias), trimmed) {
			return fmt.Errorf("name %q conflicts with alias of profile %q", trimmed, candidate.Name)
		}
	}

	return nil
}

// ValidateAlias validates a proposed profile alias. Empty alias is allowed.
// ownName is the proposed/current name of the profile receiving the alias so
// that we can detect an alias that duplicates its own profile's name.
func ValidateAlias(alias, ownName string, existing []Profile) error {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return nil
	}
	if err := checkIdentifierShape(trimmed, "alias"); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(ownName), trimmed) {
		return errors.New("alias cannot equal the profile name")
	}

	for _, candidate := range existing {
		if strings.EqualFold(candidate.Name, trimmed) {
			return fmt.Errorf("alias %q conflicts with profile %q", trimmed, candidate.Name)
		}
		if candidate.Alias != "" && strings.EqualFold(strings.TrimSpace(candidate.Alias), trimmed) {
			return fmt.Errorf("alias %q is already used by profile %q", trimmed, candidate.Name)
		}
	}

	return nil
}

func checkIdentifierShape(id, label string) error {
	if strings.Contains(id, "/") {
		return fmt.Errorf("%s cannot contain slashes", label)
	}
	if id == "." || id == ".." {
		return fmt.Errorf("%s cannot be . or ..", label)
	}
	if filepath.Clean(id) != id {
		return fmt.Errorf("%s must not contain path traversal segments", label)
	}
	if reservedIdentifiers[strings.ToLower(id)] {
		return fmt.Errorf("%s %q is reserved", label, id)
	}
	return nil
}

// FindByName looks up a profile by exact name and returns it with its index.
func FindByName(profiles []Profile, name string) (Profile, int, bool) {
	query := strings.TrimSpace(name)
	for i, candidate := range profiles {
		if strings.EqualFold(strings.TrimSpace(candidate.Name), query) {
			return candidate, i, true
		}
	}

	return Profile{}, -1, false
}

// FindByIdentifier looks up a profile by alias first, then falls back to its
// name. This matches user intent: aliases are explicit shortcuts so they win,
// and any profile lacking an alias remains reachable by its name.
func FindByIdentifier(profiles []Profile, identifier string) (Profile, int, bool) {
	query := strings.TrimSpace(identifier)
	if query == "" {
		return Profile{}, -1, false
	}
	for i, candidate := range profiles {
		if candidate.Alias != "" && strings.EqualFold(strings.TrimSpace(candidate.Alias), query) {
			return candidate, i, true
		}
	}
	return FindByName(profiles, query)
}

// Add appends a profile after validating it. An empty Path is allowed and
// signals that the agent itself manages its config directory.
func Add(profiles []Profile, candidate Profile) ([]Profile, error) {
	if err := ValidateName(candidate.Name, profiles); err != nil {
		return nil, err
	}
	if err := ValidateAlias(candidate.Alias, candidate.Name, profiles); err != nil {
		return nil, err
	}

	return append(append([]Profile(nil), profiles...), candidate), nil
}

// DeleteByName removes a profile by name.
func DeleteByName(profiles []Profile, name string) ([]Profile, error) {
	_, index, ok := FindByName(profiles, name)
	if !ok {
		return nil, fmt.Errorf("profile %q does not exist", name)
	}

	updated := append([]Profile(nil), profiles[:index]...)
	updated = append(updated, profiles[index+1:]...)
	return updated, nil
}

// Names returns the configured profile names.
func Names(profiles []Profile) []string {
	names := make([]string, 0, len(profiles))
	for _, configuredProfile := range profiles {
		names = append(names, configuredProfile.Name)
	}

	return names
}
