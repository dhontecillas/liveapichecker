package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/dhontecillas/liveapichecker/pkg/pathmatcher"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/go-openapi/loads"
)

// CoverageChecker uses an openapi specdoc and checks
// recorded api calls to keep track of what endpoints
// have been covered
type CoverageChecker struct {
	specDoc     *loads.Document
	pathMatcher *pathmatcher.PathMatcher
	basePath    string

	// covered is a map of path -> method -> status code -> covered
	covered      map[string]map[string]*EndpointCoverage
	allEndpoints []*EndpointCoverage
	rwMutex      sync.RWMutex
}

// EndpointCoverage contains the information about
// how an endpoint has been covered
type EndpointCoverage struct {
	Method                  string       `json:"method"`
	Path                    string       `json:"path"`
	StatusCodes             map[int]bool `json:"statusCodes"`
	UndocumentedStatusCodes map[int]bool `json:"undocumentedStatusCodes"`
}

// NewEndpointCoverage creates a new EndpontCoverage data
func NewEndpointCoverage(method string, path string) *EndpointCoverage {
	return &EndpointCoverage{
		Method:                  method,
		Path:                    path,
		StatusCodes:             make(map[int]bool),
		UndocumentedStatusCodes: make(map[int]bool),
	}
}

// NewCoverageChecker creates a new CoverageChecker
func NewCoverageChecker(specDoc *loads.Document) *CoverageChecker {
	bp := path.Clean(specDoc.BasePath())
	if len(bp) > 0 && bp[len(bp)-1] == '/' {
		bp = bp[:len(bp)-1]
	}
	if bp == "." {
		// if not basePath is set, it might be set to .
		bp = "/"
	}
	/*
		fmt.Printf("HOST: %s\n", specDoc.Host())
		fmt.Printf("SPEC: %#v\n", specDoc.OrigSpec())
	*/

	covered := make(map[string]map[string]*EndpointCoverage)
	pMatcher := pathmatcher.NewPathMatcher()
	ops := specDoc.Analyzer.Operations()
	for method, mops := range ops {
		for rePath, def := range mops {
			mU := strings.ToUpper(method)
			routePath := path.Join(bp, rePath)
			ec := NewEndpointCoverage(mU, routePath)
			// fmt.Printf("%s %s (bp: %s , p: %s)\n", method, routePath, bp, rePath)
			pMatcher.AddRoute(method, routePath)
			if def.Responses != nil {
				for v, _ := range def.Responses.StatusCodeResponses {
					ec.StatusCodes[v] = false
					// fmt.Printf("%d -> %#v\n", v, r)
				}
			}
			if _, ok := covered[routePath]; !ok {
				covered[routePath] = make(map[string]*EndpointCoverage)
			}
			covered[routePath][mU] = ec
		}
	}
	pMatcher.Build()

	return &CoverageChecker{
		specDoc:     specDoc,
		pathMatcher: pMatcher,
		basePath:    bp,
		covered:     covered,
	}
}

// ProcessRecordedResponse implements the RecordedRresponseProcessorHandler
// interface, and updates the stats for received request
func (cc *CoverageChecker) ProcessRecordedResponse(rwr *proxy.ResponseWriterRecorder) {
	fmt.Printf("\nanalizing request\n")

	reqPath := path.Clean(rwr.Req.URL.Path)
	matchedPath := cc.pathMatcher.LookupRoute(rwr.Req.Method, reqPath)
	if matchedPath == nil {
		fmt.Printf("No matching path for: %s  %s\n", rwr.Req.Method, reqPath)
		return
	}
	fmt.Printf("MATCHED %s -> %s\n", matchedPath.Str(), reqPath)
	cc.rwMutex.Lock()
	defer cc.rwMutex.Unlock()
	e := cc.covered[matchedPath.Path][matchedPath.Method]
	if _, ok := e.StatusCodes[rwr.StatusCode]; ok {
		e.StatusCodes[rwr.StatusCode] = true
	} else {
		e.UndocumentedStatusCodes[rwr.StatusCode] = true
	}
}

// DumpResultsToJsonString returns a report of the coverage as
// a JSON serialized string
func (cc *CoverageChecker) DumpResultsToJSONString() (string, error) {
	type report struct {
		Endpoints []*EndpointCoverage `json:"endpoints"`
	}
	var r report

	cc.rwMutex.RLock()
	for _, k := range cc.covered {
		for _, j := range k {
			r.Endpoints = append(r.Endpoints, j)
		}
	}
	cc.rwMutex.RUnlock()

	res, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// DumpResultsToFile writes the collected coverage to a file
func (cc *CoverageChecker) DumpResultsToFile(fileWithPath string) {
	f, err := os.Create(fileWithPath)
	if err != nil {
		if os.IsExist(err) {
			err = os.Remove(fileWithPath)
			if err != nil {
				fmt.Printf("cannot remove existing report file: %s", err.Error())
				return
			}
			f, err = os.Create(fileWithPath)
			if err != nil {
				fmt.Printf("cannot create report file after removal: %s", err.Error())
				return
			}
		} else {
			fmt.Printf("cannot create report file: %s", err.Error())
			return
		}
	}
	defer f.Close()

	res, err := cc.DumpResultsToJSONString()
	if err != nil {
		fmt.Printf("cannot write report: %s", err.Error())
		return
	}
	f.WriteString(string(res))
}
