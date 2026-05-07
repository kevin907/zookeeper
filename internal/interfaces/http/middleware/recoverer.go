package middleware

import (
	"fmt"
	"net/http"

	"github.com/kevin907/zookeeper/internal/interfaces/http/httperr"
	"github.com/kevin907/zookeeper/internal/platform/logging"
)

// Recoverer recovers from panics in downstream handlers, logs them at error
// level, and renders a 500 problem+json via the single httperr.Map path so
// handlers and middleware never construct Problem inline.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				logging.FromContext(r.Context()).Error("panic recovered", "panic", rv)
				// Untyped error → Map's default branch → 500 internal with a
				// scrubbed detail string.
				httperr.Write(w, httperr.Map(fmt.Errorf("panic: %v", rv)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
