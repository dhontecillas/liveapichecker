package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/dhontecillas/liveapichecker/pkg/pathmatcher"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
)

// EndpointCoverage contains the information about
// how an endpoint has been covered
type EndpointCoverage struct {
	Method                  string          `json:"method"`
	Path                    string          `json:"path"`
	StatusCodes             []int           `json:"statusCodes"`
	UndocumentedStatusCodes []int           `json:"undocumentedStatusCodes"`
	Params                  *ParamsCoverage `json:"params"`
}

type endpointCovTracker struct {
	path                    string
	statusCodes             map[int]bool
	undocumentedStatusCodes map[int]bool
	params                  *ParamsCoverageChecker
}

// newEndpointCoverage creates a new EndpointCoverage data
func newEndpointCoverageTracker(specPath string, opSpec *spec.Operation) *endpointCovTracker {
	covVariants := NewFullCoverageMinVariants()
	pcc, err := NewParamsCoverageChecker(opSpec, &covVariants)
	if err != nil {
		pcc = &ParamsCoverageChecker{}
	}
	return &endpointCovTracker{
		path:                    specPath,
		statusCodes:             make(map[int]bool),
		undocumentedStatusCodes: make(map[int]bool),
		params:                  pcc,
	}
}

// Report returns the EndpointCoverage
func (e *endpointCovTracker) Report(method string) *EndpointCoverage {
	return &EndpointCoverage{
		Method:                  method,
		Path:                    e.path,
		StatusCodes:             intSetToSlice(e.statusCodes),
		UndocumentedStatusCodes: intSetToSlice(e.undocumentedStatusCodes),
		Params:                  e.params.Report(),
	}
}

// CoverageChecker uses an openapi specdoc and checks
// recorded api calls to keep track of what endpoints
// have been covered
type CoverageChecker struct {
	rwMutex sync.RWMutex

	specDoc     *loads.Document
	pathMatcher *pathmatcher.PathMatcher
	basePath    string

	// covered is a map of path -> method -> status code -> covered
	covered map[string]map[string]*endpointCovTracker

	// allEndpoints []*EndpointCoverage
	// reportNonMatchedRequests bool
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

	covered := make(map[string]map[string]*endpointCovTracker)
	pMatcher := pathmatcher.NewPathMatcher()
	ops := specDoc.Analyzer.Operations()
	for method, mops := range ops {
		for specPath, def := range mops {
			mU := strings.ToUpper(method)
			routePath := path.Join(bp, specPath)
			ec := newEndpointCoverageTracker(specPath, def)
			pMatcher.AddRoute(method, routePath)
			if def.Responses != nil {
				for v := range def.Responses.StatusCodeResponses {
					ec.statusCodes[v] = false
				}
			}
			if _, ok := covered[routePath]; !ok {
				covered[routePath] = make(map[string]*endpointCovTracker)
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
	if _, ok := e.statusCodes[rwr.StatusCode]; ok {
		e.statusCodes[rwr.StatusCode] = true
	} else {
		e.undocumentedStatusCodes[rwr.StatusCode] = true
	}
}

func (cc *CoverageChecker) Report() []*EndpointCoverage {
	cc.rwMutex.RLock()
	eps := make([]*EndpointCoverage, len(cc.covered))
	for method, pathMap := range cc.covered {
		for _, covTrack := range pathMap {
			eps = append(eps, covTrack.Report(method))
		}
	}
	cc.rwMutex.RUnlock()
	return eps
}

// DumpResultsToJsonString returns a report of the coverage as
// a JSON serialized string
func (cc *CoverageChecker) DumpResultsToJSONString() (string, error) {
	type report struct {
		Endpoints []*EndpointCoverage `json:"endpoints"`
	}
	var r report
	r.Endpoints = cc.Report()

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

func intSetToSlice(im map[int]bool) []int {
	if len(im) == 0 {
		return []int{}
	}
	is := make([]int, 0, len(im))
	for k := range im {
		is = append(is, k)
	}
	sort.Ints(is)
	return is
}
