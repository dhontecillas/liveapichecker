package proxy

import (
	"bytes"
	"net/http"
)

type ProxyHandler struct {
	scheme     string
	forwardURL string
	client     *http.Client
}

func NewProxyHandler(forwardURL string) *ProxyHandler {
	return &ProxyHandler{
		scheme:     "http",
		forwardURL: forwardURL,
		client:     &http.Client{},
	}
}

func (ph *ProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// just write the success header for now
	// c := context.Background()
	nr, err := http.NewRequestWithContext(req.Context(),
		req.Method, req.URL.String(), req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	nr.Host = ph.forwardURL
	nr.URL.Scheme = ph.scheme
	nr.URL.Host = ph.forwardURL

	for key, slc := range req.Header {
		nr.Header[key] = make([]string, len(slc))
		copy(nr.Header[key], slc)
	}

	res, err := ph.client.Do(nr)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	dstH := rw.Header()
	for key, slc := range res.Header {
		dstH[key] = make([]string, len(slc))
		copy(dstH[key], slc)
	}
	rw.WriteHeader(res.StatusCode)

	var buf []byte
	if res.ContentLength >= 0 {
		buf = make([]byte, 0, res.ContentLength)
	}
	b := bytes.NewBuffer(buf)
	b.ReadFrom(res.Body)
	rw.Write(b.Bytes())
}
