# Define command paths
RM := /bin/rm -rf
MKDIR := /bin/mkdir -p
GO_BUILD := go build
GO_TEST := go test

# Declare phony targets that don't produce files
.PHONY: build test clean run pre-commit

# Build the export command
build:
	$(GO_BUILD) -o bin/export cmd/export/main.go

pre-commit:
	pre-commit run --all-files

# Run tests, specifically library unit tests
test:
	$(GO_TEST) ./cmd/export/...
	$(GO_TEST) ./synology_drive_api/...
	$(GO_TEST) ./synology_drive_exporter/...

test-full: test
	USE_REAL_SYNOLOGY=1 $(GO_TEST) -count=1 ./synology_drive_api/...

# Clean up automatically generated files
clean:
	$(RM) bin/export out

# Build and run the export command
run: build
	$(MKDIR) out
	./bin/export -output out
