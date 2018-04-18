package proxy

import (
	"net/http"
)

/*
ResponseWriter wraps http.ResponseWriter to provide access to the status code.
*/
type ResponseWriter struct {
	status int
	writer http.ResponseWriter
}

/*
Status returns the current HTTP status code.
*/
func (w *ResponseWriter) Status() int {
	return w.status
}

/*
Header implements http.ResponseWriter.
*/
func (w *ResponseWriter) Header() http.Header {
	return w.writer.Header()
}

/*
Write implements http.ResponseWriter.
*/
func (w *ResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

/*
WriteHeader implements http.ResponseWriter.
*/
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.writer.WriteHeader(statusCode)
}
