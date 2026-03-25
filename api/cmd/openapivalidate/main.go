// openapivalidate loads docs/openapi.yaml and checks OpenAPI 3 structural validity (kin-openapi).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-spec path]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Validates OpenAPI 3.x. Tries ../docs/openapi.yaml then docs/openapi.yaml when -spec is omitted.")
	}
	spec := flag.String("spec", "", "path to openapi.yaml (optional)")
	flag.Parse()

	path := *spec
	if path == "" {
		for _, p := range []string{"../docs/openapi.yaml", "docs/openapi.yaml"} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				path = p
				break
			}
		}
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "openapivalidate: no spec file; use -spec path/to/openapi.yaml")
		os.Exit(1)
	}

	loader := &openapi3.Loader{Context: context.Background(), IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapivalidate: load %s: %v\n", path, err)
		os.Exit(1)
	}
	if err := doc.Validate(loader.Context); err != nil {
		fmt.Fprintf(os.Stderr, "openapivalidate: validate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("openapi.yaml: OK")
}
