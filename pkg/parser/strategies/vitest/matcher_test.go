package vitest

import (
	"context"
	"testing"
)

func TestMatcher_ParseConfig(t *testing.T) {
	t.Parallel()

	m := &Matcher{}

	tests := []struct {
		name            string
		content         string
		wantGlobalsMode bool
	}{
		{
			name: "globals true",
			content: `export default defineConfig({
				test: {
					globals: true,
				}
			})`,
			wantGlobalsMode: true,
		},
		{
			name: "globals true with spaces",
			content: `export default {
				test: {
					globals :  true,
				}
			}`,
			wantGlobalsMode: true,
		},
		{
			name: "globals false",
			content: `export default defineConfig({
				test: {
					globals: false,
				}
			})`,
			wantGlobalsMode: false,
		},
		{
			name: "no globals setting",
			content: `export default defineConfig({
				test: {
					environment: 'jsdom',
				}
			})`,
			wantGlobalsMode: false,
		},
		{
			name:            "empty config",
			content:         `export default {}`,
			wantGlobalsMode: false,
		},
		{
			name: "commented globals true should be ignored",
			content: `export default defineConfig({
				test: {
					// globals: true,
					environment: 'jsdom',
				}
			})`,
			wantGlobalsMode: false,
		},
		{
			name: "globals true after comment",
			content: `export default defineConfig({
				test: {
					// some comment
					globals: true,
				}
			})`,
			wantGlobalsMode: true,
		},
		{
			name: "globals true with glob patterns containing /* and */",
			content: `export default defineConfig({
				test: {
					include: ["extension/**/*.ts", "src/**/*.ts"],
					exclude: [
						"**/node_modules/**",
						"**/dist/**",
						"view/**/*",
					],
					globals: true,
				}
			})`,
			wantGlobalsMode: true,
		},
		{
			name: "globals false with glob patterns",
			content: `export default defineConfig({
				test: {
					include: ["**/*.test.ts"],
					globals: false,
				}
			})`,
			wantGlobalsMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := m.ParseConfig(context.Background(), []byte(tt.content))
			if info == nil {
				t.Fatal("ParseConfig returned nil")
			}
			if info.Framework != frameworkName {
				t.Errorf("Framework = %q, want %q", info.Framework, frameworkName)
			}
			if info.GlobalsMode != tt.wantGlobalsMode {
				t.Errorf("GlobalsMode = %v, want %v", info.GlobalsMode, tt.wantGlobalsMode)
			}
		})
	}
}

func TestMatcher_Priority(t *testing.T) {
	t.Parallel()

	m := &Matcher{}
	if m.Priority() != matcherPriority {
		t.Errorf("Priority() = %d, want %d", m.Priority(), matcherPriority)
	}
}
