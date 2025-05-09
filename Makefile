# Define command paths
RM := /bin/rm -rf
GO_BUILD := go build
GO_TEST := go test

# Build the export command
build:
	$(GO_BUILD) -o bin/export cmd/export/main.go

# Run tests, specifically library unit tests
test:
	$(GO_TEST) ./synology_drive_api/...
	$(GO_TEST) ./synology_drive_exporter/...

# Clean up automatically generated files
clean:
	$(RM) main

# Build and run the export command
run: build
	./bin/export
