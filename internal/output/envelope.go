package output

type Envelope struct {
	OK         bool  `json:"ok"`
	Data       any   `json:"data"`
	Warning    any   `json:"warning"`
	Error      any   `json:"error"`
	DurationMS int64 `json:"duration_ms"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
