// Package tests provides integration test helpers and fixtures for the
// openmenses engine. Tests in this package exercise the full stack:
// service → validation → rules → storage.
package tests

import (
	"os"
	"path/filepath"
	"runtime"
)

// fixtureBaseDir returns the absolute path to the testdata/fixtures directory.
// It resolves relative to this source file, so it works regardless of the
// working directory when tests are run.
func fixtureBaseDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// engine/tests/ → engine/ → (repo root) → testdata/fixtures
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "fixtures")
}

// sampleExportBaseDir returns the absolute path to testdata/sample-exports/.
func sampleExportBaseDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "sample-exports")
}

// LoadFixtureBytes reads a fixture file by name from testdata/fixtures/ and
// returns its raw JSON bytes. The returned bytes are in the exportPayload
// format used by the service's ImportData RPC (version "1" JSON envelope with
// protojson-encoded records).
//
// Example:
//
//	data, err := LoadFixtureBytes("regular_28day_user.json")
//	// pass data directly to client.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: data}))
func LoadFixtureBytes(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(fixtureBaseDir(), name))
}

// LoadSampleExportBytes reads a sample export file by name from
// testdata/sample-exports/ and returns its raw JSON bytes. These files
// represent the inner payload of an ExportDataResponse and can be used to
// verify import/export round-trips.
func LoadSampleExportBytes(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(sampleExportBaseDir(), name))
}

// FixtureNames returns the names of all fixture files available in
// testdata/fixtures/. Useful for table-driven tests that should run against
// every fixture.
func FixtureNames() ([]string, error) {
	entries, err := os.ReadDir(fixtureBaseDir())
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
