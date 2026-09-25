package middleware

import "context"

func contextWithBoardRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, BoardRoleKey, role)
}

// Only set when the request passed RequireBoardMember.
func BoardRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(BoardRoleKey).(string)
	return role, ok && role != ""
}
