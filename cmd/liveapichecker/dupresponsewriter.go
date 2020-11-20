package main

import (
	"net/http"
)

type DupResponseWriter struct {
	headers           http.Header
	arw               http.ResponseWriter
	brw               http.ResponseWriter
	writeHeaderCalled bool
}

func NewDupResponseWriter(arw http.ResponseWriter, brw http.ResponseWriter) *DupResponseWriter {
	return &DupResponseWriter{
		headers: make(http.Header),
		arw:     arw,
		brw:     brw,
	}
}

func (drw *DupResponseWriter) Header() http.Header {
	return drw.headers
}

func (drw *DupResponseWriter) Write(data []byte) (int, error) {
	// WriteHeader only writes the header if it has not been
	// previously written
	drw.WriteHeader(http.StatusOK)

	a, errA := drw.arw.Write(data)
	if errA == nil {
		// writting to b, and having an error should not mean
		// that we not complete the operation on a
		drw.brw.Write(data)
	}
	return a, errA
}

func (drw *DupResponseWriter) WriteHeader(statusCode int) {
	if drw.writeHeaderCalled {
		return
	}
	drw.writeHeaderCalled = true
	drw.setHeaders()
	drw.arw.WriteHeader(statusCode)
	drw.brw.WriteHeader(statusCode)
}

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

type ResponseWriterRecorder struct {
	statusCode int
	headers    http.Header
	data       []byte

	req         *http.Request
	onWriteChan chan<- *ResponseWriterRecorder
}

func NewResponseWriterRecorder(req *http.Request, onWriteChan chan<- *ResponseWriterRecorder) *ResponseWriterRecorder {
	return &ResponseWriterRecorder{
		headers:     make(http.Header),
		req:         req,
		onWriteChan: onWriteChan,
	}
}

func (rwr *ResponseWriterRecorder) Header() http.Header {
	return rwr.headers
}

func (rwr *ResponseWriterRecorder) Write(data []byte) (int, error) {
	// WriteHeader only writes the header if it has not been
	// previously written
	rwr.WriteHeader(http.StatusOK)
	rwr.data = make([]byte, len(data))
	copy(rwr.data, data)
	rwr.onWriteChan <- rwr
	return len(data), nil
}

func (rwr *ResponseWriterRecorder) WriteHeader(statusCode int) {
	if rwr.statusCode != 0 {
		return
	}
	rwr.statusCode = statusCode
}
