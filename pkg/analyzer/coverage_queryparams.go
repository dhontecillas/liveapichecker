package analyzer

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-openapi/spec"
)

const (
	ParamInQuery  string = "query"
	ParamInPath   string = "path"
	ParamInHeader string = "header"
	ParamInBody   string = "body"
	ParamInForm   string = "form"
)

// FullCoverageMinVariants defines the number of different values
// each type of param must have to be considered fully covered.
type FullCoverageMinVariants struct {
	ParamInQuery  int
	ParamInPath   int
	ParamInHeader int
	ParamInBody   int
	ParamInForm   int
}

// FullCoverageMinVariantsFn is the function to overwrite config defaults
type FullCoverageMinVariantsFn func(cc *FullCoverageMinVariants)

// MinVariansFor return the number of different values required to
// consider a parameter fully covered
func (f *FullCoverageMinVariants) MinVariantsFor(paramIn string) int {
	switch paramIn {
	case ParamInQuery:
		return f.ParamInQuery
	case ParamInPath:
		return f.ParamInPath
	case ParamInHeader:
		return f.ParamInHeader
	case ParamInBody:
		return f.ParamInBody
	case ParamInForm:
		return f.ParamInForm
	}
	return 0
}

// NewFullCoverageMinVariants creates a configuration to define what type
// of params to include in the output result, and the number
func NewFullCoverageMinVariants(configFns ...FullCoverageMinVariantsFn) FullCoverageMinVariants {
	conf := FullCoverageMinVariants{
		ParamInQuery:  2,
		ParamInPath:   2,
		ParamInHeader: 2,
		ParamInBody:   2,
	}
	for _, cf := range configFns {
		cf(&conf)
	}
	return conf
}

// ParamCoverage reports the coverage for a given parameter for an endpoint
type ParamCoverage struct {
	Name                    string   `json:"name"`
	FullCoverageMinVariants int      `json:"full_coverage_min_variants"`
	CoveredValues           []string `json:"covered_values"`
	// EmptyAllowed            bool     `json:"empty_allowed"`
	// EmptyCovered            bool     `json:"empty_covered"`
}

// ParamsCoverage creates holds the report for the coverage
// of the query params of an endpoint
type ParamsCoverage struct {
	Params []ParamCoverage `json:"params"`
}

type paramValues struct {
	coveredSet   map[string]bool
	minVariants  int
	emptyAllowed bool
}

type ParamsCoverageChecker struct {
	byPlaceAndName map[string]map[string]*paramValues
}

func (pcc *ParamsCoverageChecker) recordInQuery(pvals map[string]*paramValues,
	r *http.Request) {
	if len(pvals) == 0 {
		return
	}
	qvals := r.URL.Query()
	for k, v := range qvals {
		if _, ok := pvals[k]; ok {
			pvals[k].coveredSet[flattenStringSlice(v)] = true
		}
	}
}

func (pcc *ParamsCoverageChecker) recordInPath(pvals map[string]*paramValues,
	r *http.Request) {
	// TODO
}

func (pcc *ParamsCoverageChecker) recordInHeader(pvals map[string]*paramValues,
	r *http.Request) {
	// TODO
}

func (pcc *ParamsCoverageChecker) recordInBody(pvals map[string]*paramValues,
	r *http.Request) {
	// TODO
}

func (pcc *ParamsCoverageChecker) recordInForm(pvals map[string]*paramValues,
	r *http.Request) {
	// TODO
}

func (pcc *ParamsCoverageChecker) Record(r *http.Request) {
	pcc.recordInQuery(pcc.byPlaceAndName[ParamInQuery], r)
	pcc.recordInPath(pcc.byPlaceAndName[ParamInPath], r)
	pcc.recordInHeader(pcc.byPlaceAndName[ParamInHeader], r)
	pcc.recordInBody(pcc.byPlaceAndName[ParamInBody], r)
	pcc.recordInForm(pcc.byPlaceAndName[ParamInForm], r)
}

func (pcc *ParamsCoverageChecker) Report() *ParamsCoverage {
	np := pcc.NumParams()
	if np == 0 {
		return &ParamsCoverage{}
	}
	params := make([]ParamCoverage, 0, np)
	for _, namedParams := range pcc.byPlaceAndName {
		for name, paramValues := range namedParams {
			covered := make([]string, 0, len(paramValues.coveredSet))
			for cv := range paramValues.coveredSet {
				covered = append(covered, cv)
			}
			params = append(params, ParamCoverage{
				Name:          name,
				CoveredValues: covered,
			})
		}
	}
	return &ParamsCoverage{
		Params: params,
	}
}

func (pcc *ParamsCoverageChecker) NumParams() int {
	cnt := 0
	for _, namedParams := range pcc.byPlaceAndName {
		cnt += len(namedParams)
	}
	return cnt
}

// NewParamsCoverageChecker
func NewParamsCoverageChecker(opSpec *spec.Operation,
	covVariants *FullCoverageMinVariants) (*ParamsCoverageChecker, error) {

	if opSpec == nil {
		return nil, fmt.Errorf("nil spec.Operation")
	}

	pcc := &ParamsCoverageChecker{
		byPlaceAndName: make(map[string]map[string]*paramValues),
	}

	for _, param := range opSpec.Parameters {
		var minCovVariants int
		if len(param.Enum) > 0 {
			minCovVariants = len(param.Enum)
		} else {
			minCovVariants = covVariants.MinVariantsFor(param.In)
		}
		if _, ok := pcc.byPlaceAndName[param.In]; !ok {
			pcc.byPlaceAndName[param.In] = make(map[string]*paramValues)
		}
		pcc.byPlaceAndName[param.In][param.Name] = &paramValues{
			coveredSet:   make(map[string]bool),
			minVariants:  minCovVariants,
			emptyAllowed: !param.Required,
		}
	}
	return pcc, nil
}

// flattenStringSlice joins different values for a single parameter
// using ','
func flattenStringSlice(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	var l int
	for _, s := range strs {
		l += len(s) + 1
	}

	sort.Strings(strs)

	sb := strings.Builder{}
	sb.Grow(l)
	for _, s := range strs {
		sb.WriteString(s)
		sb.WriteString(",")
	}
	return sb.String()
}
