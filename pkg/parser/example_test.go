package parser_test

import (
	"context"
	"fmt"
	"time"

	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/source"

	// Import strategies to register them with the default registry.
	_ "github.com/specvital/core/pkg/parser/strategies/gotesting"
	_ "github.com/specvital/core/pkg/parser/strategies/jest"
	_ "github.com/specvital/core/pkg/parser/strategies/playwright"
	_ "github.com/specvital/core/pkg/parser/strategies/vitest"
)

func Example() {
	ctx := context.Background()

	// Create a source for the project directory
	src, err := source.NewLocalSource("/path/to/project")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer src.Close()

	// Scan a project directory for test files
	result, err := parser.Scan(ctx, src)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print discovered test files
	for _, file := range result.Inventory.Files {
		fmt.Printf("File: %s (framework: %s)\n", file.Path, file.Framework)
		fmt.Printf("  Tests: %d\n", file.CountTests())
	}

	// Check for non-fatal errors
	for _, scanErr := range result.Errors {
		fmt.Printf("Warning: %v\n", scanErr)
	}
}

func Example_withOptions() {
	ctx := context.Background()

	// Create a source for the project directory
	src, err := source.NewLocalSource("/path/to/project")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer src.Close()

	// Scan with custom options
	result, err := parser.Scan(ctx, src,
		parser.WithWorkers(4),                             // Use 4 parallel workers
		parser.WithTimeout(2*time.Minute),                 // Set 2 minute timeout
		parser.WithExclude([]string{"fixtures"}),          // Skip fixtures directory
		parser.WithScanPatterns([]string{"**/*.test.ts"}), // Only *.test.ts files
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d test files\n", len(result.Inventory.Files))
}

func ExampleScan_testInventory() {
	ctx := context.Background()

	// Create a source for the project directory
	src, err := source.NewLocalSource("/path/to/project")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer src.Close()

	result, err := parser.Scan(ctx, src)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	inv := result.Inventory
	fmt.Printf("Project: %s\n", inv.RootPath)
	fmt.Printf("Total test files: %d\n", len(inv.Files))
	fmt.Printf("Total tests: %d\n", inv.CountTests())

	// Iterate test structure
	for _, file := range inv.Files {
		fmt.Printf("\n%s (%s):\n", file.Path, file.Framework)

		// Top-level tests
		for _, test := range file.Tests {
			fmt.Printf("  - %s [%d:%d]\n", test.Name, test.Location.StartLine, test.Location.EndLine)
		}

		// Test suites
		for _, suite := range file.Suites {
			fmt.Printf("  %s:\n", suite.Name)
			for _, test := range suite.Tests {
				fmt.Printf("    - %s\n", test.Name)
			}
		}
	}
}
