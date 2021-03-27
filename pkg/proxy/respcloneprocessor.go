package proxy

import (
	"net/http"
)

// RecordedResponseProcessorHandler defines an interface
// that can receive a recorded response in ResponseWriterRecorder
// and access its in-memory data.
type RecordedResponseProcessorHandler interface {
	ProcessRecordedResponse(rwr *ResponseWriterRecorder)
}

// ParallelHandler implements an http.Handler middleware
// that wraps another http.Handler and a provided
type ParallelHandler struct {
	handler             http.Handler
	parallelProcRunning bool
	parallelProc        chan *ResponseWriterRecorder
	respProcessor       RecordedResponseProcessorHandler
}

// NewParallelHandler creaete
func NewParallelHandler(h http.Handler,
	recRespProcessor RecordedResponseProcessorHandler) (p *ParallelHandler) {

	return &ParallelHandler{
		handler:       h,
		respProcessor: recRespProcessor,
	}
}

// ServeHTTP creates a new ResponseWriterRecorder, and wraps it
// as the secondary response writer for a DupResponseWriter that
// uses the original response writer as the primary one.
// Once served, the recorder is sent to the paraller processor.
func (p *ParallelHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	recRW := NewResponseWriterRecorder(req, p.parallelProc)
	dupRW := NewDupResponseWriter(rw, recRW)
	p.handler.ServeHTTP(dupRW, req)

	// send the recorded request / response to be processed
	if p.parallelProc != nil {
		p.parallelProc <- recRW
	}
}

// LaunchParallalProc launches the gorounten that will be in
// charge of processing the results
func (p *ParallelHandler) LaunchParallelProc() {
	if p.parallelProc != nil {
		return
	}

	p.parallelProc = make(chan *ResponseWriterRecorder, 300)
	go func() {
		for {
			select {
			case r := <-p.parallelProc:
				p.respProcessor.ProcessRecordedResponse(r)
			}
		}
	}()
}
