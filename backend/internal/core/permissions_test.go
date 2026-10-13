package core

import "testing"

var allRoles = []BoardRole{RoleOwner, RoleManager, RoleMember, ""}

func TestAtLeast_Matrix(t *testing.T) {
	cases := []struct {
		role, min BoardRole
		want      bool
	}{
		{RoleOwner, RoleManager, true},
		{RoleManager, RoleManager, true},
		{RoleMember, RoleManager, false},
		{RoleMember, RoleMember, true},
		{"", RoleMember, false},
		{"admin", RoleMember, false},
	}
	for _, c := range cases {
		if got := c.role.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}

func TestCanDeleteCard_AssigneeMemberCannotDelete(t *testing.T) {
	if CanDeleteCard(RoleMember, false) {
		t.Error("a member who did not create the card must not delete it")
	}
	if !CanDeleteCard(RoleMember, true) {
		t.Error("the creator must be able to delete their card")
	}
	if !CanDeleteCard(RoleManager, false) {
		t.Error("a manager must be able to delete any card")
	}
}

func TestCanEditCard_AssigneeCanEdit(t *testing.T) {
	if !CanEditCard(RoleMember, false, true) {
		t.Error("the assignee must be able to edit")
	}
	if CanEditCard(RoleMember, false, false) {
		t.Error("an unrelated member must not edit")
	}
}

func TestCanManageColumns_MemberDenied(t *testing.T) {
	want := map[BoardRole]bool{RoleOwner: true, RoleManager: true, RoleMember: false, "": false}
	for _, r := range allRoles {
		if got := CanManageColumns(r); got != want[r] {
			t.Errorf("CanManageColumns(%q) = %v, want %v", r, got, want[r])
		}
	}
}

func TestCanManageMember_Matrix(t *testing.T) {
	allowed := map[[2]BoardRole]bool{
		{RoleOwner, RoleManager}:  true,
		{RoleOwner, RoleMember}:   true,
		{RoleManager, RoleMember}: true,
	}
	for _, actor := range allRoles {
		for _, target := range allRoles {
			want := allowed[[2]BoardRole{actor, target}]
			if got := CanManageMember(actor, target); got != want {
				t.Errorf("CanManageMember(%q, %q) = %v, want %v", actor, target, got, want)
			}
		}
	}
}

func TestCanGrantRole_Matrix(t *testing.T) {
	allowed := map[[2]BoardRole]bool{
		{RoleOwner, RoleManager}:  true,
		{RoleOwner, RoleMember}:   true,
		{RoleManager, RoleMember}: true,
	}
	for _, actor := range allRoles {
		for _, role := range allRoles {
			want := allowed[[2]BoardRole{actor, role}]
			if got := CanGrantRole(actor, role); got != want {
				t.Errorf("CanGrantRole(%q, %q) = %v, want %v", actor, role, got, want)
			}
		}
	}
}

func TestCanDeletePlanningSession_Matrix(t *testing.T) {
	cases := []struct {
		role                BoardRole
		isCreator, hasItems bool
		want                bool
	}{
		{RoleMember, true, false, true},
		{RoleMember, true, true, false},
		{RoleMember, false, false, false},
		{RoleManager, false, true, true},
		{RoleOwner, false, true, true},
	}
	for _, c := range cases {
		if got := CanDeletePlanningSession(c.role, c.isCreator, c.hasItems); got != c.want {
			t.Errorf("CanDeletePlanningSession(%q, creator=%v, items=%v) = %v, want %v",
				c.role, c.isCreator, c.hasItems, got, c.want)
		}
	}
}
