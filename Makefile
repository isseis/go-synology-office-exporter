# Define command paths
RM := /bin/rm -rf
MKDIR := /bin/mkdir -p
GO_BUILD := go build
GO_TEST := go test

# Declare phony targets that don't produce files
.PHONY: build test test-unit test-full test-race clean run pre-commit

# Build the export command
build:
	$(GO_BUILD) -o bin/export cmd/export/main.go

pre-commit:
	pre-commit run --all-files

test: test-unit

# Run unit tests (excludes integration tests)
test-unit:
	$(GO_TEST) -tags test ./...

# Run integration tests (requires Synology NAS environment variables)
test-integration:
	$(GO_TEST) -tags test,integration -count=1 ./synology_drive_api/...

# Run race tests (to find race conditions)
test-race:
	$(GO_TEST) -tags test,race -race -v -timeout=5s -count=1 ./download_history/...

# Run all tests (unit + integration + race)
test-full: test-unit test-integration test-race

# Clean up automatically generated files
clean:
	$(RM) bin/export out

# Build and run the export command
run: build
	$(MKDIR) out
	./bin/export -output out
