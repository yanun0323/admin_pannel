package model

// Switcher represents a trading pair switcher configuration
// The document structure is dynamic with trading pair keys (e.g., "SOL_USDT")
type Switcher struct {
	MongoID string                  `json:"id"`
	Pairs   map[string]SwitcherPair `json:"pairs"`
}

// SwitcherPair represents the enable status for a trading pair
type SwitcherPair struct {
	Enable bool `json:"enable" bson:"enable"`
}

// SwitcherResponse is the API response structure
type SwitcherResponse struct {
	ID    string                  `json:"id"`
	Pairs map[string]SwitcherPair `json:"pairs"`
}

// UpdateSwitcherRequest is the request structure for updating a switcher
type UpdateSwitcherRequest struct {
	Pairs map[string]SwitcherPair `json:"pairs"`
}

// ToResponse converts Switcher to SwitcherResponse
func (s *Switcher) ToResponse() SwitcherResponse {
	return SwitcherResponse{
		ID:    s.MongoID,
		Pairs: s.Pairs,
	}
}
