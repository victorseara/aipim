package profile

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	existing := []Profile{{Name: "work"}}

	cases := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"empty", "", "empty"},
		{"whitespace", "   ", "empty"},
		{"slash", "a/b", "slash"},
		{"dot", ".", "."},
		{"dotdot", "..", "."},
		{"duplicate case-insensitive", "WORK", "already exists"},
		{"valid", "personal", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.input, existing)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestAddAndDelete(t *testing.T) {
	profiles := []Profile{}

	added, err := Add(profiles, Profile{Name: "a", Path: "/tmp/a"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if len(added) != 1 || added[0].Name != "a" {
		t.Fatalf("unexpected profiles after add: %+v", added)
	}

	if _, err := Add(added, Profile{Name: "a", Path: "/tmp/a2"}); err == nil {
		t.Fatal("expected duplicate-name error")
	}

	// Empty path is allowed: signals the agent manages its own config dir.
	withEmptyPath, err := Add(added, Profile{Name: "agent-managed", Path: ""})
	if err != nil {
		t.Fatalf("Add with empty path should succeed: %v", err)
	}
	if len(withEmptyPath) != 2 {
		t.Fatalf("expected 2 profiles after empty-path Add, got %d", len(withEmptyPath))
	}

	deleted, err := DeleteByName(added, "A")
	if err != nil {
		t.Fatalf("DeleteByName (case-insensitive) failed: %v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected empty profiles after delete, got %+v", deleted)
	}

	if _, err := DeleteByName(added, "missing"); err == nil {
		t.Fatal("expected error deleting nonexistent profile")
	}
}

func TestValidateNameRejectsReserved(t *testing.T) {
	for _, reserved := range []string{"launch", "create", "help", "completion", "LAUNCH"} {
		if err := ValidateName(reserved, nil); err == nil {
			t.Errorf("ValidateName(%q) should reject reserved word", reserved)
		}
	}
}

// TestValidateNameForEditingAllowsUnchanged verifies the self-edit pattern:
// callers pass the "other profiles" slice (excluding the one being edited) so
// keeping the same name is allowed.
func TestValidateNameForEditingAllowsUnchanged(t *testing.T) {
	all := []Profile{
		{Name: "work", Alias: "w"},
		{Name: "personal"},
	}

	// Editing "work" with name unchanged: filter it out and re-validate.
	others := append([]Profile(nil), all[1:]...)
	if err := ValidateName("work", others); err != nil {
		t.Fatalf("renaming self to its current name should be allowed, got: %v", err)
	}

	// But renaming "personal" → "work" while "work" still exists must fail.
	others = append([]Profile(nil), all[:1]...)
	if err := ValidateName("work", others); err == nil {
		t.Fatal("renaming to an existing name should fail")
	}
}

func TestValidateNameRejectsAliasCollision(t *testing.T) {
	existing := []Profile{{Name: "work", Alias: "w"}}
	err := ValidateName("w", existing)
	if err == nil {
		t.Fatal("ValidateName should reject a name that matches an existing alias")
	}
	if !strings.Contains(err.Error(), "alias") {
		t.Errorf("expected error to mention alias collision, got: %v", err)
	}
}

func TestValidateAlias(t *testing.T) {
	existing := []Profile{
		{Name: "work", Alias: "w"},
		{Name: "personal"},
	}

	cases := []struct {
		name    string
		alias   string
		ownName string
		wantErr string
	}{
		{"empty allowed", "", "new", ""},
		{"slash", "a/b", "new", "slash"},
		{"reserved", "launch", "new", "reserved"},
		{"matches own name", "new", "new", "equal the profile name"},
		{"matches existing name", "work", "new", "conflicts with profile"},
		{"matches existing alias", "w", "new", "already used"},
		{"matches existing name case-insensitively", "WORK", "new", "conflicts with profile"},
		{"valid fresh alias", "n", "new", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAlias(tc.alias, tc.ownName, existing)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestFindByIdentifierBackcompat is the migration guarantee: profiles WITHOUT
// an alias remain launchable via their name through `aipim <name>`. Adding the
// alias feature must not break older configs.
//
// Lookup priority is alias first, then name. Validation prevents alias/name
// collisions across profiles, so in practice both orderings resolve the same
// profile — alias-first just expresses "alias is the explicit shortcut."
func TestFindByIdentifierBackcompat(t *testing.T) {
	profiles := []Profile{
		{Name: "sgws"},                  // legacy: no alias
		{Name: "work", Alias: "w"},      // new style
		{Name: "personal", Alias: "p"},
	}

	cases := map[string]string{
		"sgws":     "sgws",     // legacy name lookup still works (no alias)
		"SGWS":     "sgws",     // case-insensitive
		"work":     "work",     // name fallback resolves
		"w":        "work",     // alias hit
		"personal": "personal",
		"p":        "personal",
	}

	for query, wantName := range cases {
		t.Run(query, func(t *testing.T) {
			got, _, ok := FindByIdentifier(profiles, query)
			if !ok {
				t.Fatalf("FindByIdentifier(%q) returned ok=false", query)
			}
			if got.Name != wantName {
				t.Errorf("FindByIdentifier(%q) = %q, want %q", query, got.Name, wantName)
			}
		})
	}

	if _, _, ok := FindByIdentifier(profiles, "ghost"); ok {
		t.Fatal("FindByIdentifier should not match an unknown identifier")
	}
	if _, _, ok := FindByIdentifier(profiles, ""); ok {
		t.Fatal("FindByIdentifier should not match an empty query")
	}
}

func TestFindByNameAndNames(t *testing.T) {
	profiles := []Profile{{Name: "Alpha"}, {Name: "Beta"}}

	got, idx, ok := FindByName(profiles, "alpha")
	if !ok || idx != 0 || got.Name != "Alpha" {
		t.Fatalf("FindByName failed: ok=%v idx=%d got=%+v", ok, idx, got)
	}

	if _, _, ok := FindByName(profiles, "missing"); ok {
		t.Fatal("expected missing profile lookup to fail")
	}

	names := Names(profiles)
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Fatalf("Names returned %+v", names)
	}
}

func TestAddDoesNotMutateInput(t *testing.T) {
	original := []Profile{{Name: "a"}}
	added, err := Add(original, Profile{Name: "b", Path: "/tmp/b"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if len(original) != 1 {
		t.Fatalf("input slice mutated: len=%d", len(original))
	}
	if len(added) != 2 {
		t.Fatalf("expected 2 profiles in result, got %d", len(added))
	}
}
