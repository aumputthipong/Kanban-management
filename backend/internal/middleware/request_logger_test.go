package middleware

import "testing"

func TestIsSensitivePath(t *testing.T) {
	t.Parallel()
	// Note: callers pass r.URL.Path which never contains the query string,
	// so we only assert on bare paths.
	cases := []struct {
		path string
		want bool
	}{
		{"/api/auth/google/callback", true},
		{"/api/auth/google/callback/extra", true}, // prefix match — sub-paths still redact
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
