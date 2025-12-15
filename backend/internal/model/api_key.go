package model

import "time"

type Platform string

const (
	PlatformBinance Platform = "binance"
	PlatformBTCC    Platform = "btcc"
	PlatformOKX     Platform = "okx"
	PlatformBybit   Platform = "bybit"
)

func (p Platform) String() string {
	return string(p)
}

func (p Platform) IsValid() bool {
	switch p {
	case PlatformBinance, PlatformBTCC, PlatformOKX, PlatformBybit:
		return true
	default:
		return false
	}
}

func AllPlatforms() []Platform {
	return []Platform{
		PlatformBinance,
		PlatformBTCC,
		PlatformOKX,
		PlatformBybit,
	}
}

type APIKey struct {
	ID        string    `json:"id"` // MongoDB ObjectID as string
	Name      string    `json:"name"`
	Platform  Platform  `json:"platform"`
	APIKey    string    `json:"api_key"`
	APISecret string    `json:"-"` // Never expose in JSON responses
	IsTestnet bool      `json:"is_testnet"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIKeyResponse is the response structure that masks sensitive data
type APIKeyResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Platform        Platform  `json:"platform"`
	APIKeyMasked    string    `json:"api_key_masked"`
	APISecretMasked string    `json:"api_secret_masked"`
	IsTestnet       bool      `json:"is_testnet"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ToResponse converts APIKey to APIKeyResponse with masked sensitive data
func (a *APIKey) ToResponse() APIKeyResponse {
	return APIKeyResponse{
		ID:              a.ID,
		Name:            a.Name,
		Platform:        a.Platform,
		APIKeyMasked:    maskAPIKey(a.APIKey),
		APISecretMasked: maskAPIKey(a.APISecret),
		IsTestnet:       a.IsTestnet,
		IsActive:        a.IsActive,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}

// maskAPIKey masks the API key, showing only first 4 and last 4 characters
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// CreateAPIKeyRequest is the request structure for creating an API key
type CreateAPIKeyRequest struct {
	Name      string   `json:"name"`
	Platform  Platform `json:"platform"`
	APIKey    string   `json:"api_key"`
	APISecret string   `json:"api_secret"`
	IsTestnet bool     `json:"is_testnet"`
}

// UpdateAPIKeyRequest is the request structure for updating an API key
type UpdateAPIKeyRequest struct {
	Name      *string `json:"name,omitempty"`
	APIKey    *string `json:"api_key,omitempty"`
	APISecret *string `json:"api_secret,omitempty"`
	IsTestnet *bool   `json:"is_testnet,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}
