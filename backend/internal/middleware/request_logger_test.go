package middleware

import "testing"

func TestIsSensitivePath(t *testing.T) {
	t.Parallel()
	// Callers pass r.URL.Path, which never has a query string.
	cases := []struct {
		path string
		want bool
	}{
		{"/api/auth/google/callback", true},
		{"/api/auth/google/callback/extra", true},
		{"/api/auth/login", false},
		{"/api/auth/google", false},
		{"/api/boards", false},
		{"/", false},
	}
	for _, c := range cases {
		if got := isSensitivePath(c.path); got != c.want {
			t.Errorf("isSensitivePath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
