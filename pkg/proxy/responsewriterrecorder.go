package proxy

import (
	"bytes"
	"net/http"
)

// ResponseWriterRecorder implements an http.ResponseWriter
// that stores written headers and data in memory, along with
// a pointer to the original request, so the request / response
// can be inspected.
type ResponseWriterRecorder struct {
	Req        *http.Request // reference to the original request
	StatusCode int
	Headers    http.Header
	Data       *bytes.Buffer
}

// NewResponseWriterRecorder creates new ResponseWriterRecorder
func NewResponseWriterRecorder(req *http.Request,
	onWriteFinishedChan chan<- *ResponseWriterRecorder) *ResponseWriterRecorder {
	// clones the request, so it can fiddle with it in a separate goroutine
	// when main processing completes
	clonedReq := req.Clone(req.Context())
	return &ResponseWriterRecorder{
		Req:        clonedReq,
		StatusCode: 0,
		Headers:    make(http.Header),
		Data:       new(bytes.Buffer),
	}
}

// Header returns the http.Header for the the response
func (rwr *ResponseWriterRecorder) Header() http.Header {
	return rwr.Headers
}

// Write writes the data in the response
func (rwr *ResponseWriterRecorder) Write(data []byte) (int, error) {
	// WriteHeader only writes the header if it has not been
	// previously written
	rwr.WriteHeader(http.StatusOK)
	if rwr.Data == nil {
		rwr.Data = bytes.NewBuffer(data)
	} else {
		rwr.Data.Write(data)
	}
	return len(data), nil
}

// WriteHeader sets the status code
func (rwr *ResponseWriterRecorder) WriteHeader(statusCode int) {
	if rwr.StatusCode != 0 {
		return
	}
	rwr.StatusCode = statusCode
}
