IMAGE:=databus23/power-metrics

.PHONY: build

build:
	GOOS=linux CGO_ENABLED=0 go build -o build/power-metrics -ldflags="-s -w" github.com/databus23/power-metrics
	docker build -t $(IMAGE) .

push:
	docker push $(IMAGE)

