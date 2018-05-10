package proxy

import (
	"net/http"
)

/*
ResponseWriter implements http.ResponseWriter and is used to intercept
the responses from the proxied services to capture and inspect 502 and
503 errors.
*/
type ResponseWriter struct {
	data   []byte
	header http.Header
	status int
}

/*
Status returns the current response status code.
*/
func (w *ResponseWriter) Status() int {
	return w.status
}

/*
Header implements http.ResponseWriter and captures response header data.
*/
func (w *ResponseWriter) Header() http.Header {
	return w.header
}

/*
Write implements http.ResponseWriter and captures response body data.
*/
func (w *ResponseWriter) Write(data []byte) (int, error) {
	w.data = append(w.data, data...)
	return len(data), nil
}

/*
WriteHeader implements http.ResponseWriter and captures response status
codes.
*/
func (w *ResponseWriter) WriteHeader(code int) {
	w.status = code
}
