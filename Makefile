VERSION ?= v0.3

docker:
	docker build -t dhontecillas/liveapichecker:latest -t dhontecillas/liveapichecker:$(VERSION) .
	docker image prune --filter label=stage=builder

dockeralpine:
	docker build -f Dockerfile.alpine -t dhontecillas/liveapichecker-alpine:latest -t dhontecillas/liveapichecker-alpine:$(VERSION) .
	docker image prune --filter label=stage=builder

dockerpushalpine:
	docker push dhontecillas/liveapichecker-alpine:latest
	docker push dhontecillas/liveapichecker-alpine:$(VERSION)
