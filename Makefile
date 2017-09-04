#
# A Makefile that builds gochat in a build container and can also launch the
# dev. environment & tail logs from it.
#
# Author: Étienne Lafarge <etienne.lafarge _at_ gmail.com>
#
BUILD_CONTAINER = \
  docker run -u $(shell id -u) --rm \
	  --workdir "/usr/local/go/src/github.com/elafarge/gochat" \
	  -v $(shell pwd):/usr/local/go/src/github.com/elafarge/gochat:ro \
	  -v $(shell pwd)/vendor:/vendor/src \
	  -e GOPATH="/go:/vendor" \
	  -e CGO_ENABLED=0 \
	  -e GOOS=linux

GLIDE_CONTAINER = \
	docker run --rm \
	  --workdir "/usr/local/go/src/github.com/elafarge/gochat" \
	  -v $$(pwd):/usr/local/go/src/github.com/elafarge/gochat \
		$(BUILD_CONTAINER_IMAGE)

BUILD_CONTAINER_IMAGE = golang:1

GOBUILD = go build --installsuffix cgo --ldflags '-extldflags \"-static\"'

.PHONY: bin bin-clean clean vendor-clean vendor-update dev devlog devdown

# Default target
bin: main/build/target
clean: bin-clean vendor-clean

# 1. Vendor
vendor: glide.yaml
	@echo "Pulling dependencies with glide... in a build container"
	rm -rf ./vendor
	mkdir ./vendor
	$(GLIDE_CONTAINER) bash -c \
		"go get github.com/Masterminds/glide && glide install && chown $(shell id -u):$(shell id -g) -R ./glide.lock ./vendor"

vendor-update:
	@echo "Pulling dependencies with glide... in a build container"
	$(GLIDE_CONTAINER) bash -c \
		"go get github.com/Masterminds/glide && glide update && chown $(shell id -u):$(shell id -g) -R ./glide.lock ./vendor"

vendor-clean:
	@echo "Dropping the vendor folder"
	rm -rf ./vendor

# 2. Build
bin-clean:
	@echo "Deleting binary"
	rm -rf main/build

main/build/target: main/*.go broker/*.go *.go vendor
	@echo "Building binary"
	mkdir -p main/build
	$(BUILD_CONTAINER) -v $(shell pwd)/main/build:/build:rw $(BUILD_CONTAINER_IMAGE) \
		$(GOBUILD) -o /build/target ./main

# 3. And dev.
dev: main/build/target
	@echo "Starting dev. env. (ports: front=4690, back=4691)"
	docker-compose up -d

devlog:
	docker-compose logs --follow

devdown:
	docker-compose down
