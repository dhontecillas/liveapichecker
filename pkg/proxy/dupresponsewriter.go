package proxy

import (
	"net/http"
)

// DupResponseWriter is an http.ResponseWriter that just
// writes response to two different http.ResponseWriter
// that have been provided at construction time.
type DupResponseWriter struct {
	headers           http.Header
	arw               http.ResponseWriter
	brw               http.ResponseWriter
	writeHeaderCalled bool
}

// NewDupResponseWriter creates a new DupResponseWriter
// that will write to two provided ResponseWriter s.
// The first response writer is treated as the main one,
// and errors returned by it, will be the ones returned
// by the created ResponseWriter, however, if an error
// happens when writting to the second one, that would be
// ignored.
func NewDupResponseWriter(arw http.ResponseWriter, brw http.ResponseWriter) *DupResponseWriter {
	return &DupResponseWriter{
		headers: make(http.Header),
		arw:     arw,
		brw:     brw,
	}
}

// Header returns the headers written to the DupResponseWriter
func (drw *DupResponseWriter) Header() http.Header {
	return drw.headers
}

// Write writes data to both wrapped http.Response writers.
// Headers will be written if that wasn't previously done.
// If an error happend when writting to the primary wrapped
// writer, that will be returned, and writting to the secondary
// one won't be attempted. However, if it succeeds, the
// write to the secondary is not checked for errors (so it
// could fail silently).
func (drw *DupResponseWriter) Write(data []byte) (int, error) {
	drw.WriteHeader(http.StatusOK)

	a, errA := drw.arw.Write(data)
	if errA == nil {
		// writting to b, and having an error should not mean
		// that we not complete the operation on a
		drw.brw.Write(data)
	}
	return a, errA
}

// WriteHeader makes the headers to be written to both
// wrapped http.ResponseWriter
func (drw *DupResponseWriter) WriteHeader(statusCode int) {
	if drw.writeHeaderCalled {
		return
	}
	drw.writeHeaderCalled = true
	drw.setHeaders()
	drw.arw.WriteHeader(statusCode)
	drw.brw.WriteHeader(statusCode)
}

// setHeaders writes the current headers to both wrapped
// http.ResponseWriter s
func (drw *DupResponseWriter) setHeaders() {
	aH := drw.arw.Header()
	bH := drw.brw.Header()
	for k, s := range drw.headers {
		acs := make([]string, 0, len(s))
		bcs := make([]string, 0, len(s))
		for _, v := range s {
			acs = append(acs, v)
			bcs = append(bcs, v)
		}
		aH[k] = acs
		bH[k] = bcs
	}
}
