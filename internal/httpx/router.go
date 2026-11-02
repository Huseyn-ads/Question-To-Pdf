package httpx

import "net/http"

// JSONRouter preserves ServeMux's route matching and Allow header, while
// returning the same JSON error shape for unknown paths and wrong methods.
func JSONRouter(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		result := &routeErrorWriter{header: make(http.Header), status: http.StatusNotFound}
		handler.ServeHTTP(result, r)
		if allow := result.header.Get("Allow"); allow != "" {
			w.Header().Set("Allow", allow)
		}
		_ = WriteError(w, result.status, http.StatusText(result.status))
	})
}

type routeErrorWriter struct {
	header http.Header
	status int
}

func (w *routeErrorWriter) Header() http.Header            { return w.header }
func (w *routeErrorWriter) WriteHeader(status int)         { w.status = status }
func (w *routeErrorWriter) Write(body []byte) (int, error) { return len(body), nil }
