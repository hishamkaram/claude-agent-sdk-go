package types

// SettingsResult represents the current applied settings for a Claude Code session.
type SettingsResult struct {
	Applied AppliedSettings        `json:"applied"`
	Raw     map[string]interface{} `json:"raw,omitempty"`
}

// AppliedSettings contains the structured settings currently in effect.
type AppliedSettings struct {
	Model  string `json:"model"`
	Effort string `json:"effort"`
}
