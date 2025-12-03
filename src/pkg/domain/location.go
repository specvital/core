package domain

type Location struct {
	EndCol    int    `json:"endCol,omitempty"`
	EndLine   int    `json:"endLine"`
	File      string `json:"file"`
	StartCol  int    `json:"startCol,omitempty"`
	StartLine int    `json:"startLine"`
}
