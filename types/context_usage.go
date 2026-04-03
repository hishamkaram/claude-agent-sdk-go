package types

// ContextUsage represents the token usage breakdown for the current session context.
type ContextUsage struct {
	TotalTokens    int                      `json:"totalTokens"`
	MaxTokens      int                      `json:"maxTokens"`
	UtilizationPct float64                  `json:"utilizationPct"`
	ByCategory     map[string]CategoryUsage `json:"byCategory,omitempty"`
}

// CategoryUsage represents token usage for a single category within the context.
type CategoryUsage struct {
	Tokens int     `json:"tokens"`
	Pct    float64 `json:"pct"`
}
