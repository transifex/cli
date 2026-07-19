#!/bin/sh

# -coverprofile writes a machine-readable profile (coverage.out) for SonarQube
# (sonar.go.coverage.reportPaths); -covermode=atomic keeps counts correct under
# concurrent tests. Profile entries are module-import paths, resolved by the
# scanner from the module root (sonar.sources=.).
go test -covermode=atomic -coverprofile=coverage.out ./...
