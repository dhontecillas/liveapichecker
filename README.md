# Live API Check

Middleware to check that requests / response comply with OpenAPI
spec, and to collects stats and coverage of endpoints and responses.


## Environment vars configuration


```bash

LIVEAPICHECKER_OPENAPI_FILE="/path/to/openapi/file"
LIVEAPICHECKER_REPORT_FILE="/path/to/report/file"
LIVEAPICHECKER_FORWARD_URL="http://localhost:8000"
LIVEAPICHECKER_FORWARD_LISTENAT="locahost:7777"

```
