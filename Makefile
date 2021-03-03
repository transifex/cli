PLATFORM=local

build:
	DOCKER_BUILDKIT=1 docker build --output bin/ --platform ${PLATFORM} --target bin .

