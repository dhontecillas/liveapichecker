package proxy

import (
	"bytes"
	"net/http"
	"net/url"
)

// ProxyHandler implements the http.Handler interface
type ProxyHandler struct {
	scheme string
	host   string
	client *http.Client
}

// NewProxyHandler creates a new ProxyHandler with a url to
// forward requests to
func NewProxyHandler(forwardURL string) (*ProxyHandler, error) {
	fwURL, err := url.Parse(forwardURL)
	if err != nil {
		return nil, err
	}
	return &ProxyHandler{
		scheme: fwURL.Scheme,
		host:   fwURL.Host,
		client: &http.Client{},
	}, nil
}

// ServeHTTP is the main handler for proxying requests.
// Reads a request, creates a new one changing its Host
// and Scheme to the configured ones, and then just
// writes the response to the http.ResponseWriter
func (ph *ProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	nr, err := http.NewRequestWithContext(req.Context(),
		req.Method, req.URL.String(), req.Body)

	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	nr.Host = ph.host
	nr.URL.Scheme = ph.scheme
	nr.URL.Host = ph.host
	nr.ContentLength = req.ContentLength

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
