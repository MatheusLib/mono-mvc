package middleware

import "net/http"

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func newStatusWriter(w http.ResponseWriter) *statusWriter {
	return &statusWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func ensureStatusWriter(w http.ResponseWriter) (*statusWriter, http.ResponseWriter) {
	if sw, ok := w.(*statusWriter); ok {
		return sw, w
	}
	sw := newStatusWriter(w)
	return sw, sw
}
