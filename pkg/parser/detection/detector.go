package detection

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/detection/extraction"
	"github.com/specvital/core/pkg/parser/framework"
)

// Detector performs confidence-based framework detection.
// It uses a multi-stage detection algorithm that evaluates:
// 1. Config scope matches (highest confidence: 80 points)
// 2. Import statements (high confidence: 60 points)
// 3. Content patterns (moderate confidence: 40 points)
// 4. Filename patterns (low confidence: 20 points)
//
// Evidence is accumulated across all stages, and the framework
// with the highest total confidence (without negative evidence) is selected.
type Detector struct {
	registry     *framework.Registry
	projectScope *framework.AggregatedProjectScope
}

// NewDetector creates a new confidence-based detector.
func NewDetector(registry *framework.Registry) *Detector {
	return &Detector{
		registry:     registry,
		projectScope: nil,
	}
}

// SetProjectScope configures the detector with project-wide configuration context.
// This enables scope-based detection (highest confidence).
func (d *Detector) SetProjectScope(scope *framework.AggregatedProjectScope) {
	d.projectScope = scope
}

// Detect performs multi-stage framework detection on a test file.
func (d *Detector) Detect(ctx context.Context, filePath string, content []byte) Result {
	lang := detectLanguage(filePath)
	if lang == "" {
		return Unknown()
	}

	frameworks := d.registry.FindByLanguage(lang)
	if len(frameworks) == 0 {
		return Unknown()
	}

	results := make(map[string]*Result)

	d.detectFromScope(filePath, lang, results)
	d.detectFromImports(ctx, lang, content, frameworks, results)
	d.detectFromContent(ctx, content, frameworks, results)
	d.detectFromFilename(ctx, filePath, frameworks, results)

	return d.selectBestMatch(results)
}

func (d *Detector) detectFromScope(filePath string, lang domain.Language, results map[string]*Result) {
	if d.projectScope == nil {
		return
	}

	type scopeMatch struct {
		path  string
		scope *framework.ConfigScope
		depth int
	}

	var matches []scopeMatch
	for path, scope := range d.projectScope.Configs {
		def := d.registry.Find(scope.Framework)
		if def == nil {
			continue
		}

		langCompatible := false
		for _, l := range def.Languages {
			if l == lang {
				langCompatible = true
				break
			}
		}
		if !langCompatible {
			continue
		}

		if scope.Contains(filePath) {
			matches = append(matches, scopeMatch{
				path:  path,
				scope: scope,
				depth: scope.Depth(),
			})
		}
	}

	if len(matches) == 0 {
		return
	}

	best := matches[0]
	for _, m := range matches[1:] {
		if m.depth > best.depth {
			best = m
		}
	}

	result := getOrCreate(results, best.scope.Framework)
	result.Scope = best.scope
	result.AddEvidence(Evidence{
		Source:      "config-scope",
		Description: "File matches config scope: " + best.path,
		Confidence:  80,
	})

	if best.scope.GlobalsMode {
		result.AddEvidence(Evidence{
			Source:      "globals-mode",
			Description: "Config has globals: true",
			Confidence:  10,
		})
	}
}

func (d *Detector) detectFromImports(ctx context.Context, lang domain.Language, content []byte, frameworks []*framework.Definition, results map[string]*Result) {
	var imports []string

	switch lang {
	case domain.LanguageTypeScript, domain.LanguageJavaScript:
		imports = extraction.ExtractJSImports(ctx, content)
	case domain.LanguageGo:
		return
	}

	if len(imports) == 0 {
		return
	}

	for _, fw := range frameworks {
		for _, matcher := range fw.Matchers {
			for _, imp := range imports {
				signal := framework.Signal{
					Type:  framework.SignalImport,
					Value: imp,
				}

				mr := matcher.Match(ctx, signal)
				if mr.Confidence > 0 || mr.Negative {
					result := getOrCreate(results, fw.Name)
					for _, ev := range mr.Evidence {
						result.AddEvidence(Evidence{
							Source:      "import",
							Description: ev,
							Confidence:  60,
							Negative:    mr.Negative,
						})
					}
				}
			}
		}
	}
}

func (d *Detector) detectFromContent(ctx context.Context, content []byte, frameworks []*framework.Definition, results map[string]*Result) {
	for _, fw := range frameworks {
		for _, matcher := range fw.Matchers {
			signal := framework.Signal{
				Type:    framework.SignalFileContent,
				Value:   "",
				Context: content,
			}

			mr := matcher.Match(ctx, signal)
			if mr.Confidence > 0 || mr.Negative {
				result := getOrCreate(results, fw.Name)
				for _, ev := range mr.Evidence {
					result.AddEvidence(Evidence{
						Source:      "content",
						Description: ev,
						Confidence:  40,
						Negative:    mr.Negative,
					})
				}
			}
		}
	}
}

func (d *Detector) detectFromFilename(ctx context.Context, filePath string, frameworks []*framework.Definition, results map[string]*Result) {
	filename := filepath.Base(filePath)

	for _, fw := range frameworks {
		for _, matcher := range fw.Matchers {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: filename,
			}

			mr := matcher.Match(ctx, signal)
			if mr.Confidence > 0 || mr.Negative {
				result := getOrCreate(results, fw.Name)
				for _, ev := range mr.Evidence {
					result.AddEvidence(Evidence{
						Source:      "filename",
						Description: ev,
						Confidence:  20,
						Negative:    mr.Negative,
					})
				}
			}
		}
	}
}

func (d *Detector) selectBestMatch(results map[string]*Result) Result {
	hasNegative := make(map[string]bool)
	hasImport := make(map[string]bool)

	for fw, result := range results {
		for _, ev := range result.Evidence {
			if ev.Negative {
				hasNegative[fw] = true
			}
			if ev.Source == "import" && !ev.Negative {
				hasImport[fw] = true
			}
		}
	}

	var best Result
	bestHasImport := false

	for fw, result := range results {
		if hasNegative[fw] {
			continue
		}

		total := 0
		for _, ev := range result.Evidence {
			if !ev.Negative {
				total += ev.Confidence
			}
		}

		if total > 100 {
			total = 100
		}

		result.Framework = fw
		result.Confidence = total

		currentHasImport := hasImport[fw]
		if currentHasImport && !bestHasImport {
			best = *result
			bestHasImport = true
		} else if currentHasImport == bestHasImport && total > best.Confidence {
			best = *result
			bestHasImport = currentHasImport
		}
	}

	return best
}

func getOrCreate(results map[string]*Result, framework string) *Result {
	if r, ok := results[framework]; ok {
		return r
	}

	r := &Result{
		Framework: framework,
		Evidence:  make([]Evidence, 0, 4),
	}
	results[framework] = r
	return r
}

func detectLanguage(filePath string) domain.Language {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".ts", ".tsx":
		return domain.LanguageTypeScript
	case ".js", ".jsx", ".mjs", ".cjs":
		return domain.LanguageJavaScript
	case ".go":
		return domain.LanguageGo
	default:
		return ""
	}
}
