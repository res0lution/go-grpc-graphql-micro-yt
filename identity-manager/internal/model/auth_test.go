package model

import "testing"

func TestUserInfo_GroupHelpers(t *testing.T) {
	u := UserInfo{Group: []string{"A", "B"}}

	if !u.HasGroup("A") {
		t.Fatalf("expected HasGroup")
	}
	if u.HasGroup("C") {
		t.Fatalf("unexpected HasGroup")
	}
	if !u.HasAnyGroup([]string{"X", "B"}) {
		t.Fatalf("expected HasAnyGroup")
	}
	if !u.HasAllGroups([]string{"A", "B"}) {
		t.Fatalf("expected HasAllGroups")
	}
	if u.HasAllGroups([]string{"A", "C"}) {
		t.Fatalf("unexpected HasAllGroups")
	}

	if u.HasAnyGroup(nil) {
		t.Fatalf("unexpected HasAnyGroup for nil groups")
	}
	if !u.HasAllGroups(nil) {
		// keep existing behavior: nil required groups means no explicit restriction
		t.Fatalf("expected HasAllGroups to be true for nil groups")
	}
}

func TestUserInfo_GroupHelpers_EmptyUserGroups(t *testing.T) {
	u := UserInfo{}

	if u.HasGroup("A") {
		t.Fatalf("unexpected HasGroup")
	}
	if u.HasAnyGroup([]string{"A"}) {
		t.Fatalf("unexpected HasAnyGroup")
	}
	if u.HasAllGroups([]string{"A"}) {
		t.Fatalf("unexpected HasAllGroups")
	}
}
