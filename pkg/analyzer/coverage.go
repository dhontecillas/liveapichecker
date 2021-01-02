package analyzer

import (
	"fmt"
	"os"
	"path"

	"github.com/dhontecillas/liveapichecker/pkg/pathmatcher"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/go-openapi/loads"
)

type CoverageChecker struct {
	specDoc     *loads.Document
	pathMatcher *pathmatcher.PathMatcher
	basePath    string
}

func NewCoverageChecker(specDoc *loads.Document) *CoverageChecker {
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

	return &CoverageChecker{
		specDoc:     specDoc,
		pathMatcher: pMatcher,
		basePath:    bp,
	}
}

func (cc *CoverageChecker) ProcessRecordedResponse(rwr *proxy.ResponseWriterRecorder) {
	fmt.Printf("\nanalizing request\n")

	reqPath := path.Clean(rwr.Req.URL.Path)
	matchPath := cc.pathMatcher.LookupRoute(rwr.Req.Method, reqPath)
	if len(matchPath) == 0 {
		fmt.Printf("No matching path for: %s\n", reqPath)
		return
	}
	fmt.Printf("MATCHED %s -> %s\n", matchPath, reqPath)
}

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

	f.WriteString("Report!")

}
