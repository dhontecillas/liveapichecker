package pathmatcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-openapi/runtime/middleware/denco"
)

// MatchedPath contains the information of an endpoint matched
type MatchedPath struct {
	Method string
	Path   string
	Params map[string]string
}

// Str returns a representation of the endpoint called
func (mp *MatchedPath) Str() string {
	return fmt.Sprintf("%s %s (%v)", mp.Method, mp.Path, mp.Params)
}

// PathMatcher holds the data to match endpoint routes
// with paths defined in the OpenAPI v2 format
type PathMatcher struct {
	pathConverter *regexp.Regexp
	records       map[string][]denco.Record
	routers       map[string]*denco.Router
}

// NewPathMatcher creates a new PathMatcher
func NewPathMatcher() *PathMatcher {
	pathConverter := regexp.MustCompile(`{(.+?)}([^/]*)`)
	return &PathMatcher{
		pathConverter: pathConverter,
		records:       make(map[string][]denco.Record),
	}
}

// AddRoute adds an endpoint to be matched: method (http verb)
// and path (in OpenAPI v2 format)
func (pm *PathMatcher) AddRoute(method, path string) {
	mn := strings.ToUpper(method)
	conv := pm.pathConverter.ReplaceAllString(path, ":$1")
	record := denco.NewRecord(conv, path)
	pm.records[mn] = append(pm.records[mn], record)
}

// LookupRoute searches for an endpoint match in the PathMatcher
// and returns it in case is found (otherwise, null is returned)
func (pm *PathMatcher) LookupRoute(method, pathWithParams string) *MatchedPath {
	method = strings.ToUpper(method)
	r, ok := pm.routers[method]
	if !ok {
		return nil
	}
	res, params, found := r.Lookup(pathWithParams)
	if !found {
		return nil
	}
	str, ok := res.(string)
	if !ok {
		return nil
	}

	mp := make(map[string]string, len(params))
	for _, p := range params {
		mp[p.Name] = mp[p.Value]
	}
	return &MatchedPath{
		Method: method,
		Path:   str,
		Params: mp,
	}
}

// Build must be called before any rouute can be looked up
func (pm *PathMatcher) Build() {
	pm.routers = make(map[string]*denco.Router)
	for method, records := range pm.records {
		router := denco.New()
		_ = router.Build(records)
		pm.routers[method] = router
	}
}
