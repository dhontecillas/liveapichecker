package analyzer_test

import (
	"net/http"
	"testing"

	"github.com/dhontecillas/liveapichecker/pkg/analyzer"
	"github.com/go-openapi/loads"
	"github.com/stretchr/testify/require"
)

func TestCoverageQueryParams(t *testing.T) {
	r := require.New(t)

	doc, err := loads.Analyzed([]byte(testSwagger), "2.0")

	r.Nil(err, "swagger file")

	spec := doc.OrigSpec()
	r.NotNil(spec, "spec not nil")
	r.NotNil(spec.Paths, "spec.Paths not nil")

	op := spec.Paths.Paths["/pets"].Get
	r.NotNil(op, "cannot find endpoint to test")

	fcmv := analyzer.FullCoverageMinVariants{}
	pcc, err := analyzer.NewParamsCoverageChecker(op, &fcmv)
	r.Nil(err, "cannot create NewParamsCoverageChecker")

	nParams := pcc.NumParams()
	r.Equal(nParams, 1, "want 2 params, got %d", nParams)

	req, _ := http.NewRequest("GET", "https://example.com/pets/?limit=5", nil)
	pcc.Record(req)

	report := pcc.Report()
	r.NotNil(report)

	r.Equal(len(report.Params), 1, "should have covered the limit param")
	r.Equal(report.Params[0].Name, "limit")
	r.Equal(report.Params[0].CoveredValues[0], "5")
}
