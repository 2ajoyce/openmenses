//go:build ignore

// gen_export.go imports full_featured_user.json into an in-memory engine and
// prints the resulting export payload to stdout. Run from repo root:
//
//	go run ./tools/scripts/gen_export.go > testdata/sample-exports/full_featured_user_export.json
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"connectrpc.com/connect"
	"github.com/2ajoyce/openmenses/engine/pkg/openmenses"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

func main() {
	ctx := context.Background()
	eng, err := openmenses.NewEngine(ctx, openmenses.WithInMemory())
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewEngine: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close() //nolint:errcheck

	mux := http.NewServeMux()
	path, handler := eng.Handler()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := openmensesv1connect.NewCycleTrackerServiceClient(srv.Client(), srv.URL)

	// Import the fixture.
	data, err := os.ReadFile("testdata/fixtures/full_featured_user.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ReadFile: %v\n", err)
		os.Exit(1)
	}
	importResp, err := client.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{
		Parent: "users/full",
		Data:   data,
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateDataImport: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Imported %d records\n", importResp.Msg.GetRecordsImported())

	// Export the data.
	exportResp, err := client.CreateDataExport(ctx, connect.NewRequest(&v1.CreateDataExportRequest{
		Parent: "user-full",
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateDataExport: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print the export payload.
	var pretty interface{}
	if err := json.Unmarshal(exportResp.Msg.GetData(), &pretty); err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshal export: %v\n", err)
		os.Exit(1)
	}
	out, err := json.MarshalIndent(pretty, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "MarshalIndent: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
