package analyzer_test

import (
	"net/http"
	"testing"

	"github.com/dhontecillas/liveapichecker/pkg/analyzer"
	"github.com/dhontecillas/liveapichecker/pkg/proxy"
	"github.com/go-openapi/loads"
	"github.com/stretchr/testify/require"
)

func TestCoverage(t *testing.T) {
	r := require.New(t)
	doc, err := loads.Analyzed([]byte(testSwagger), "2.0")
	r.Nil(err, "swagger file")

	req, _ := http.NewRequest("GET", "https://example.com/pets/?limit=5", nil)
	rwr := proxy.NewResponseWriterRecorder(req)
	rwr.StatusCode = 200

	cc := analyzer.NewCoverageChecker(doc)
	cc.ProcessRecordedResponse(rwr)
	report := cc.Report()

	r.Equal(len(report), 1, "want 1 params, got %d", len(report))

	/*
		pcc.Record(req)

		report := pcc.Report()
		r.NotNil(report)

		r.Equal(len(report.Params), 1, "should have covered the limit param")
		r.Equal(report.Params[0].Name, "limit")
		r.Equal(report.Params[0].CoveredValues[0], "5")
	*/
}
