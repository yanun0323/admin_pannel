package model

// Setting represents a strategy setting configuration
type Setting struct {
	MongoID    string                 `json:"id"`
	Base       string                 `json:"base"`
	Quote      string                 `json:"quote"`
	Strategy   string                 `json:"strategy"`
	Parameters map[string]interface{} `json:"parameters"`
}

// SettingResponse is the API response structure
type SettingResponse struct {
	ID         string                 `json:"id"`
	Base       string                 `json:"base"`
	Quote      string                 `json:"quote"`
	Strategy   string                 `json:"strategy"`
	Parameters map[string]interface{} `json:"parameters"`
}

// CreateSettingRequest is the request structure for creating a setting
type CreateSettingRequest struct {
	Base       string                 `json:"base"`
	Quote      string                 `json:"quote"`
	Strategy   string                 `json:"strategy"`
	Parameters map[string]interface{} `json:"parameters"`
}

// UpdateSettingRequest is the request structure for updating a setting
type UpdateSettingRequest struct {
	Base       *string                 `json:"base,omitempty"`
	Quote      *string                 `json:"quote,omitempty"`
	Strategy   *string                 `json:"strategy,omitempty"`
	Parameters map[string]interface{}  `json:"parameters,omitempty"`
}

// ToResponse converts Setting to SettingResponse
func (s *Setting) ToResponse() SettingResponse {
	return SettingResponse{
		ID:         s.MongoID,
		Base:       s.Base,
		Quote:      s.Quote,
		Strategy:   s.Strategy,
		Parameters: s.Parameters,
	}
}
