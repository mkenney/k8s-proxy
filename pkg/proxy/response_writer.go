package proxy

import (
	"net/http"
)

/*
ResponseWriter wraps http.ResponseWriter instances to add a Status
method.
*/
type ResponseWriter struct {
	writer http.ResponseWriter
	status int
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
	return w.writer.Header()
}

/*
Write implements http.ResponseWriter
*/
func (w *ResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

/*
WriteHeader implements http.ResponseWriter
*/
func (w *ResponseWriter) WriteHeader(code int) {
	w.status = code
	w.writer.WriteHeader(code)
}
