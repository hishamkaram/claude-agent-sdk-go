package types

import "encoding/json"

// ContextUsage represents the token usage breakdown for the current session
// context. The CLI control response shape has drifted between camelCase and
// snake_case over time, so decoding accepts both while marshaling preserves the
// existing camelCase SDK wire format.
type ContextUsage struct {
	TotalTokens    int                      `json:"-"`
	MaxTokens      int                      `json:"-"`
	UtilizationPct float64                  `json:"-"`
	ByCategory     map[string]CategoryUsage `json:"-"`
}

type contextUsageWire struct {
	TotalTokens    int                      `json:"totalTokens"`
	MaxTokens      int                      `json:"maxTokens"`
	UtilizationPct float64                  `json:"utilizationPct"`
	ByCategory     map[string]CategoryUsage `json:"byCategory,omitempty"`
}

type contextUsageDecode struct {
	TotalTokensCamel *int                     `json:"totalTokens"`
	TotalTokensSnake *int                     `json:"total_tokens"`
	MaxTokensCamel   *int                     `json:"maxTokens"`
	MaxTokensSnake   *int                     `json:"max_tokens"`
	UtilPctCamel     *float64                 `json:"utilizationPct"`
	UtilPctSnake     *float64                 `json:"utilization_pct"`
	ByCategoryCamel  map[string]CategoryUsage `json:"byCategory"`
	ByCategorySnake  map[string]CategoryUsage `json:"by_category"`
}

func (u ContextUsage) MarshalJSON() ([]byte, error) {
	return json.Marshal(contextUsageWire{
		TotalTokens:    u.TotalTokens,
		MaxTokens:      u.MaxTokens,
		UtilizationPct: u.UtilizationPct,
		ByCategory:     u.ByCategory,
	})
}

func (u *ContextUsage) UnmarshalJSON(data []byte) error {
	var raw contextUsageDecode
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var decoded ContextUsage
	if raw.TotalTokensCamel != nil {
		decoded.TotalTokens = *raw.TotalTokensCamel
	} else if raw.TotalTokensSnake != nil {
		decoded.TotalTokens = *raw.TotalTokensSnake
	}
	if raw.MaxTokensCamel != nil {
		decoded.MaxTokens = *raw.MaxTokensCamel
	} else if raw.MaxTokensSnake != nil {
		decoded.MaxTokens = *raw.MaxTokensSnake
	}
	if raw.UtilPctCamel != nil {
		decoded.UtilizationPct = *raw.UtilPctCamel
	} else if raw.UtilPctSnake != nil {
		decoded.UtilizationPct = *raw.UtilPctSnake
	}
	if raw.ByCategoryCamel != nil {
		decoded.ByCategory = raw.ByCategoryCamel
	} else if raw.ByCategorySnake != nil {
		decoded.ByCategory = raw.ByCategorySnake
	}

	*u = decoded
	return nil
}

// CategoryUsage represents token usage for a single category within the context.
type CategoryUsage struct {
	Tokens int     `json:"tokens"`
	Pct    float64 `json:"pct"`
}
