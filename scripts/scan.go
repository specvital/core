//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/specvital/core/pkg/parser"
	"github.com/specvital/core/pkg/source"

	_ "github.com/specvital/core/pkg/parser/strategies/all"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/scan.go <path>\n")
		os.Exit(1)
	}

	path := os.Args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	src, err := source.NewLocalSource(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "source error: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	result, err := parser.Scan(ctx, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}

	output := map[string]interface{}{
		"filesScanned": result.Stats.FilesScanned,
		"filesMatched": result.Stats.FilesMatched,
		"testCount":    result.Inventory.CountTests(),
		"duration":     result.Stats.Duration.String(),
		"frameworks":   countFrameworks(result),
	}
	json.NewEncoder(os.Stdout).Encode(output)
}

func countFrameworks(result *parser.ScanResult) map[string]int {
	counts := make(map[string]int)
	for _, file := range result.Inventory.Files {
		counts[file.Framework]++
	}
	return counts
}
