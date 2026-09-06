package middleware

import "net/http"

// CORS allows a single trusted origin and answers preflight directly. Wildcards are
// unsupported on purpose: with Access-Control-Allow-Credentials true, browsers reject
// `*` anyway. Wired outermost in main.go so preflight returns before auth or rate limits.
func CORS(frontendURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		// Cache preflight for 10 minutes. Handlers set their own Content-Type;
		// forcing application/json here mis-typed Swagger assets and HTML error pages.
		w.Header().Set("Access-Control-Max-Age", "600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
