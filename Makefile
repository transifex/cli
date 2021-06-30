PLATFORM=local
TAG=$$(git rev-parse --short HEAD)

build: local-build

tests: cli-build cli-tests

cli-build:
	DOCKER_BUILDKIT=1 docker build -f ./build/Dockerfile --platform ${PLATFORM} -t go-cli:devel .

cli-tests:
	docker-compose run --rm cli-test

docker-build:
	DOCKER_BUILDKIT=1 docker build -f ./build/Dockerfile-bin --output bin/ --platform ${PLATFORM} --target bin .

local-build:
	go build -o ./bin/tx

local-mod-download:
	go mod download

local-test:
	./scripts/tests.sh

local-test-windows:
	.\scripts\tests.cmd
