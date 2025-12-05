package jest

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
			name: "default config (globals enabled by default)",
			content: `module.exports = {
				testEnvironment: 'node',
			}`,
			wantGlobalsMode: true,
		},
		{
			name: "injectGlobals explicitly true",
			content: `module.exports = {
				injectGlobals: true,
			}`,
			wantGlobalsMode: true,
		},
		{
			name: "injectGlobals false",
			content: `module.exports = {
				injectGlobals: false,
			}`,
			wantGlobalsMode: false,
		},
		{
			name: "injectGlobals false with spaces",
			content: `module.exports = {
				injectGlobals :  false,
			}`,
			wantGlobalsMode: false,
		},
		{
			name:            "empty config",
			content:         `module.exports = {}`,
			wantGlobalsMode: true, // Jest defaults to globals enabled
		},
		{
			name: "JSON config",
			content: `{
				"testEnvironment": "node"
			}`,
			wantGlobalsMode: true,
		},
		{
			name: "commented injectGlobals false should be ignored",
			content: `module.exports = {
				// injectGlobals: false,
				testEnvironment: 'node',
			}`,
			wantGlobalsMode: true,
		},
		{
			name: "injectGlobals false after comment",
			content: `module.exports = {
				// some comment
				injectGlobals: false,
			}`,
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
