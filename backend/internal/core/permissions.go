package core

// Single source for board permission rules; frontend mirrors this in lib/permissions.ts.

func (r BoardRole) rank() int {
	switch r {
	case RoleOwner:
		return 3
	case RoleManager:
		return 2
	case RoleMember:
		return 1
	}
	return 0
}

func (r BoardRole) AtLeast(min BoardRole) bool {
	return r.rank() > 0 && r.rank() >= min.rank()
}

func CanEditCard(role BoardRole, isCreator, isAssignee bool) bool {
	return isCreator || isAssignee || role.AtLeast(RoleManager)
}

// Assignees can edit but not delete: deletion is destructive.
func CanDeleteCard(role BoardRole, isCreator bool) bool {
	return isCreator || role.AtLeast(RoleManager)
}

func CanManageColumns(role BoardRole) bool {
	return role.AtLeast(RoleManager)
}

// Remove or re-role target. Managers act on members only; nobody acts on the owner.
func CanManageMember(actor, target BoardRole) bool {
	switch actor {
	case RoleOwner:
		return target == RoleManager || target == RoleMember
	case RoleManager:
		return target == RoleMember
	}
	return false
}

// Owner is never grantable here; it moves only by ownership transfer.
func CanGrantRole(actor, role BoardRole) bool {
	switch actor {
	case RoleOwner:
		return role == RoleManager || role == RoleMember
	case RoleManager:
		return role == RoleMember
	}
	return false
}

// Deleting a session cascades every item in it, so a creator may only drop an empty one.
func CanDeletePlanningSession(role BoardRole, isCreator, hasItems bool) bool {
	return role.AtLeast(RoleManager) || (isCreator && !hasItems)
}
