package proxy

import (
	"net/http"
)

/*
ResponseWriter wraps http.ResponseWriter instances to add a Status
method.
*/
type ResponseWriter struct {
	status int
	data   []byte
	header http.Header
}

/*
Status returns the current response status code.
*/
func (w *ResponseWriter) Status() int {
	return w.status
}

/*
Header implements http.ResponseWriter
*/
func (w *ResponseWriter) Header() http.Header {
	return w.header
}

/*
Write implements http.ResponseWriter
*/
func (w *ResponseWriter) Write(data []byte) (int, error) {
	w.data = append(w.data, data...)
	return len(data), nil
}

/*
WriteHeader implements http.ResponseWriter
*/
func (w *ResponseWriter) WriteHeader(code int) {
	w.status = code
}
