IMAGE:=databus23/power-metrics
VERSION:=0.2

.PHONY: build

build:
	GOOS=linux CGO_ENABLED=0 go build -o build/power-metrics -ldflags="-s -w" github.com/databus23/power-metrics
	docker build -t $(IMAGE):$(VERSION) .

push:
	docker push $(IMAGE):$(VERSION)

