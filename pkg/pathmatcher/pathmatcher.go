package pathmatcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-openapi/runtime/middleware/denco"
)

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
	method = strings.ToUpper(method)
	r, ok := pm.routers[method]
	if !ok {
		fmt.Printf("routers %#v\n", pm.routers)
		return ""
	}
	res, params, found := r.Lookup(pathWithParams)
	if !found {
		fmt.Printf("Lookup NOT found: %s -> %s, %#v, %t\n%#v\n",
			pathWithParams, res, params, found, pm.routers[method])
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
