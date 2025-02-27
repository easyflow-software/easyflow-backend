package turnstile

type CloudflareTurnstileResponse struct {
	Success            bool     `json:"success"`
	ChallangeTimestamp int      `json:"challange_ts,omitempty"`
	Hostname           string   `json:"hostname,omitempty"`
	ErrorCodes         []string `json:"error-codes"`
	Action             string   `json:"action,omitempty"`
	Cdata              string   `json:"cdata,omitempty"`
}
