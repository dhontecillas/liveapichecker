package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware/denco"
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
	fmt.Printf("\n\nspec:\n")

	// ops := specDoc.Analyzer.Operations()

	// MOVE: this shit inside NewParallelHandler
	pMatcher := NewPathMatcher()
	/*
		paths = specDoc.Analyzer.AllPaths()
			for k, v := range paths {
				fmt.Printf("%s -> %#v\n\n", k, v)
				for _, pi := range v.PathItemProps {
					if
				}
			}
	*/
	ops := specDoc.Analyzer.Operations()
	for method, mops := range ops {
		for path, op := range mops {
			fmt.Printf("%s %s -> %#v\n\n", method, path, op)
			pMatcher.AddRoute(method, path)
		}
	}
	pMatcher.Build()

	testMatch := pMatcher.LookupRoute("GET", "/recipients/foo")
	fmt.Printf("TMatch: %s\n", testMatch)

	proxyH := NewProxyHandler()
	parallelH := NewParallelHandler(proxyH, specDoc)
	parallelH.LaunchParallelProc()

	//		launchServer(parallelH)
}

type ProxyHandler struct {
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{}
}

func (ph *ProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// just write the success header for now
	rw.Write(([]byte)("{\"success\": true}"))
}

type ParallelHandler struct {
	handler http.Handler

	parallelProcRunning bool
	parallelProc        chan *ResponseWriterRecorder

	specDoc *loads.Document
}

func NewParallelHandler(h http.Handler, specDoc *loads.Document) (p *ParallelHandler) {
	return &ParallelHandler{
		handler: h,
		specDoc: specDoc,
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
	fmt.Printf("rwr %#v\n", rwr)

}

func launchServer(hfn http.Handler) {
	srv := &http.Server{
		// TODO: load this from config:
		Addr:    "localhost:9876",
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

type PathMatcher struct {
	pathConverter *regexp.Regexp
	records       map[string][]denco.Record
	routers       map[string]*denco.Router
}

func NewPathMatcher() *PathMatcher {
	pathConverter := regexp.MustCompile(`{(.+?)}([^/]*)`)
	return &PathMatcher{
		pathConverter: pathConverter,
		records:       make(map[string][]denco.Record),
	}
}

func (pm *PathMatcher) AddRoute(method, path string) {
	mn := strings.ToUpper(method)
	conv := pm.pathConverter.ReplaceAllString(path, ":$1")
	record := denco.NewRecord(conv, path)
	pm.records[mn] = append(pm.records[mn], record)
}

func (pm *PathMatcher) LookupRoute(method, pathWithParams string) string {
	r, ok := pm.routers[method]
	if !ok {
		return ""
	}
	res, params, found := r.Lookup(pathWithParams)
	if !found {
		fmt.Printf("Lookup NOT found\n")
		return ""
	}
	fmt.Printf("Lookup Params:\n%#v\n", params)
	str, ok := res.(string)
	if !ok {
		return ""
	}
	return str
}

func (pm *PathMatcher) Build() {
	pm.routers = make(map[string]*denco.Router)
	for method, records := range pm.records {
		router := denco.New()
		_ = router.Build(records)
		pm.routers[method] = router
	}
}
