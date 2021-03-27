FROM golang:1.15.2-buster as builder
LABEL stage=builder

COPY . /go/src/github.com/dhontecillas/liveapichecker
WORKDIR /go/src/github.com/dhontecillas/liveapichecker
RUN go build -ldflags="-w -s" ./cmd/liveapichecker

FROM bitnami/minideb:stretch
RUN install_packages ca-certificates tzdata
COPY --from=builder /go/src/github.com/dhontecillas/liveapichecker/liveapichecker /liveapichecker
VOLUME /data

ENV LIVEAPICHECKER_OPENAPI_FILE="/data/openapi.v2.yaml"
ENV LIVEAPICHECKER_REPORT_FILE="/data/report.out.json"
ENV LIVEAPICHECKER_FORWARD_TO="http://localhost:7776"
ENV LIVEAPICHECKER_FORWARD_LISTENAT="0.0.0.0:7777"
ENV LIVEAPICHECKER_REPORT_SERVER_ADDRESS="0.0.0.0:7778"

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="liveapichecker" \
      org.label-schema.description="Live API Checker" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/dhontecillas/liveapichecker" \
      org.label-schema.vcs-url="https://github.com/dhontecillas/liveapichecker" \
      org.label-schema.vcs-ref=$BUILD_VCS_REF \
      org.label-schema.vendor="David Hontecillas" \
      org.label-schema.version=$BUILD_VERSION

ENTRYPOINT "/liveapichecker"
