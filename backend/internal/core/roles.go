// Package core holds dependency-free domain types, importable from anywhere.
package core

// BoardRole maps 1:1 to board_members.role.
type BoardRole string

const (
	RoleOwner   BoardRole = "owner"
	RoleManager BoardRole = "manager"
	RoleMember  BoardRole = "member"
)

func (r BoardRole) IsValid() bool {
	switch r {
	case RoleOwner, RoleManager, RoleMember:
		return true
	}
	return false
}
