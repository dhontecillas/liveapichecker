package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/dhontecillas/liveapichecker/pkg/pathmatcher"
	"github.com/spf13/viper"

	"github.com/go-openapi/loads"
	/*
		"github.com/go-openapi/analysis"
		"github.com/go-openapi/errors"
		"github.com/go-openapi/loads"
		"github.com/go-openapi/spec"
		"github.com/go-openapi/strfmt"

		"github.com/go-openapi/runtime"
		"github.com/go-openapi/runtime/logger"
		"github.com/go-openapi/runtime/middleware/untyped"
		"github.com/go-openapi/runtime/security"
	*/)

const (
	OpenAPIFileKey string = "openapi.file"
	ForwardURLKey  string = "forward.url"
)

func main() {
	v := viper.GetViper()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	fileName := v.GetString(OpenAPIFileKey)
	if len(fileName) == 0 {
		panic("cannot read OPENAPI_FILE filename")
	}

	forwardURL := v.GetString(ForwardURLKey)
	if len(forwardURL) == 0 {
		panic("cannot read forward url")
	}
	fmt.Printf("checking %s against %s\n", fileName, forwardURL)

	specDoc, err := loads.Spec(fileName)
	if err != nil {
		fmt.Printf("\nerror in provided spec:\n%s\n", err.Error())
		return
	}

	proxyH := NewProxyHandler(forwardURL)
	parallelH := NewParallelHandler(proxyH, specDoc)
	parallelH.LaunchParallelProc()

	launchServer(parallelH)
}

type ProxyHandler struct {
	forwardURL string
	client     *http.Client
}

func NewProxyHandler(forwardURL string) *ProxyHandler {
	return &ProxyHandler{
		forwardURL: forwardURL,
		client:     &http.Client{},
	}
}

func (ph *ProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// just write the success header for now
	// c := context.Background()
	fmt.Printf("forwarding request\n")
	nr, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
	if err != nil {
		fmt.Printf("error creating request: %s\n", err.Error())
		return
	}
	// TODO: check the error
	nr.Host = ph.forwardURL
	nr.URL.Scheme = "http"
	nr.URL.Host = ph.forwardURL

	for key, slc := range req.Header {
		nr.Header[key] = make([]string, len(slc))
		copy(nr.Header[key], slc)
	}

	res, err := ph.client.Do(nr)
	if err != nil {
		fmt.Printf("error proxying the request: %s\n", err.Error())
		rw.WriteHeader(500)
		return
	}
	dstH := rw.Header()
	// srcH := res.Header
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

type ParallelHandler struct {
	handler http.Handler

	parallelProcRunning bool
	parallelProc        chan *ResponseWriterRecorder

	specDoc *loads.Document

	pathMatcher *pathmatcher.PathMatcher
	basePath    string
}

func NewParallelHandler(h http.Handler, specDoc *loads.Document) (p *ParallelHandler) {
	bp := path.Clean(specDoc.BasePath())
	if len(bp) > 0 && bp[len(bp)-1] == '/' {
		bp = bp[:len(bp)-1]
	}
	fmt.Printf("HOST: %s\n", specDoc.Host())
	fmt.Printf("SPEC: %#v\n", specDoc.OrigSpec())

	pMatcher := pathmatcher.NewPathMatcher()
	ops := specDoc.Analyzer.Operations()
	for method, mops := range ops {
		for rePath, _ := range mops {
			routePath := path.Join(bp, rePath)
			fmt.Printf("%s %s (bp: %s , p: %s)\n", method, routePath, bp, rePath)
			pMatcher.AddRoute(method, routePath)
		}
	}
	pMatcher.Build()
	return &ParallelHandler{
		handler:     h,
		specDoc:     specDoc,
		pathMatcher: pMatcher,
		basePath:    bp,
	}
}

func (p *ParallelHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	clonedReq := req.Clone(req.Context())
	recRW := NewResponseWriterRecorder(clonedReq, p.parallelProc)
	dupRW := NewDupResponseWriter(rw, recRW)
	p.handler.ServeHTTP(dupRW, req)
}

func (p *ParallelHandler) LaunchParallelProc() {
	if p.parallelProc != nil {
		return
	}

	p.parallelProc = make(chan *ResponseWriterRecorder)
	go func() {
		// var r *ResponseWriterRecorder
		for {
			select {
			case r := <-p.parallelProc:
				p.Analyze(r)
			}
		}
	}()
}

func (p *ParallelHandler) Analyze(rwr *ResponseWriterRecorder) {
	fmt.Printf("\nanalizing request\n")

	reqPath := path.Clean(rwr.req.URL.Path)
	matchPath := p.pathMatcher.LookupRoute(rwr.req.Method, reqPath)
	if len(matchPath) == 0 {
		fmt.Printf("No matching path for: %s\n", reqPath)
		return
	}
	fmt.Printf("MATCHED %s -> %s\n", matchPath, reqPath)

}

func launchServer(hfn http.Handler) {
	srv := &http.Server{
		// TODO: load this from config:
		Addr:    "0.0.0.0:7777",
		Handler: hfn,
	}

	sigChan := make(chan os.Signal, 1)

	go signalHandler(sigChan, srv)

	if err := srv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("error %s\nSHUTTING DOWN", err.Error())
		}
	}
}

func signalHandler(sc chan os.Signal, srv *http.Server) {
	signal.Notify(sc, os.Interrupt)
	<-sc
	fmt.Printf("\nshutdown signal received\n")
	shutdownServer(srv)
}

func shutdownServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

type ReqResp struct {
}

func openapiChecker() {
	select {}
}
