package domain

// Location represents a position in source code.
type Location struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	StartCol  int    `json:"startCol,omitempty"`
	EndCol    int    `json:"endCol,omitempty"`
}
