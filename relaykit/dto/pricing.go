package dto

import "github.com/QuantumNous/new-api/relaykit/types"

// 这里不好动就不动了，本来想独立出来的（
type OpenAIModels struct {
	Id                     string                     `json:"id"`
	Object                 string                     `json:"object"`
	Created                int64                      `json:"created"`
	OwnedBy                string                     `json:"owned_by"`
	SupportedEndpointTypes []types.EndpointType       `json:"supported_endpoint_types"`
	SchemaVersion          string                     `json:"schema_version,omitempty"`
	Name                   string                     `json:"name,omitempty"`
	Description            string                     `json:"description,omitempty"`
	InputModalities        []OpenRouterInputModality  `json:"input_modalities,omitempty"`
	OutputModalities       []OpenRouterOutputModality `json:"output_modalities,omitempty"`
	Pricing                []OpenRouterPrice          `json:"pricing,omitempty"`
	OpenRouter             *OpenRouterMapping         `json:"openrouter,omitempty"`
}

type OpenRouterMapping struct {
	Slug string `json:"slug"`
}

type OpenRouterInputModality struct {
	Type                  string                          `json:"type"`
	SupportedInputs       map[string]OpenRouterCapability `json:"supported_inputs,omitempty"`
	Pricing               []OpenRouterPrice               `json:"pricing,omitempty"`
	PassthroughParameters map[string]OpenRouterCapability `json:"passthrough_parameters,omitempty"`
}

type OpenRouterOutputModality struct {
	Type                  string                          `json:"type"`
	Streaming             *bool                           `json:"streaming,omitempty"`
	MaxLength             *OpenRouterLimit                `json:"max_length,omitempty"`
	SupportedParameters   map[string]OpenRouterCapability `json:"supported_parameters"`
	Pricing               []OpenRouterPrice               `json:"pricing,omitempty"`
	PassthroughParameters map[string]OpenRouterCapability `json:"passthrough_parameters,omitempty"`
}

type OpenRouterLimit struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type OpenRouterCapability struct {
	Type       string                          `json:"type,omitempty"`
	Min        *float64                        `json:"min,omitempty"`
	Max        *float64                        `json:"max,omitempty"`
	Unit       string                          `json:"unit,omitempty"`
	Values     []any                           `json:"values,omitempty"`
	Items      *OpenRouterCapability           `json:"items,omitempty"`
	MaxItems   *int                            `json:"max_items,omitempty"`
	Properties map[string]OpenRouterCapability `json:"properties,omitempty"`
	Default    any                             `json:"default,omitempty"`
	Value      *float64                        `json:"value,omitempty"`
}

type OpenRouterPrice struct {
	Type       string                    `json:"type"`
	Unit       string                    `json:"unit"`
	CostUSD    string                    `json:"cost_usd"`
	Overrides  []OpenRouterPriceOverride `json:"overrides,omitempty"`
	TTLSeconds *int                      `json:"ttl_seconds,omitempty"`
	Implicit   bool                      `json:"implicit,omitempty"`
	UTCStart   *int                      `json:"utc_start,omitempty"`
	UTCEnd     *int                      `json:"utc_end,omitempty"`
	UTCDays    []string                  `json:"utc_days,omitempty"`
}

type OpenRouterPriceOverride struct {
	When    map[string]any `json:"when"`
	CostUSD string         `json:"cost_usd"`
}

type AnthropicModel struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type GeminiModel struct {
	Name                       any   `json:"name"`
	BaseModelId                any   `json:"baseModelId"`
	Version                    any   `json:"version"`
	DisplayName                any   `json:"displayName"`
	Description                any   `json:"description"`
	InputTokenLimit            any   `json:"inputTokenLimit"`
	OutputTokenLimit           any   `json:"outputTokenLimit"`
	SupportedGenerationMethods []any `json:"supportedGenerationMethods"`
	Thinking                   any   `json:"thinking"`
	Temperature                any   `json:"temperature"`
	MaxTemperature             any   `json:"maxTemperature"`
	TopP                       any   `json:"topP"`
	TopK                       any   `json:"topK"`
}
