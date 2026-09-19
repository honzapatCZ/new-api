package dto

import "github.com/QuantumNous/new-api/relaykit/types"

// OpenAIModels is the OpenRouter provider-monitor model document. The endpoint
// intentionally emits schema 2.4 typed modalities rather than the legacy flat
// pricing object.
type OpenAIModels struct {
	SchemaVersion          string               `json:"schema_version"`
	Id                     string               `json:"id"`
	Name                   string               `json:"name"`
	Created                int64                `json:"created"`
	OwnedBy                string               `json:"owned_by,omitempty"`
	Description            string               `json:"description,omitempty"`
	SupportedEndpointTypes []types.EndpointType `json:"supported_endpoint_types,omitempty"`
	InputModalities        []OpenRouterModality `json:"input_modalities"`
	OutputModalities       []OpenRouterModality `json:"output_modalities"`
	Pricing                []OpenRouterPricingEntry `json:"pricing,omitempty"`
	Capacity               []OpenRouterCapacityEntry `json:"capacity,omitempty"`
	PassthroughParameters  map[string]any       `json:"passthrough_parameters,omitempty"`
	Datacenters            []OpenRouterDatacenter `json:"datacenters,omitempty"`
	DeploymentRegion       string               `json:"deployment_region,omitempty"`
	Compliance             *OpenRouterCompliance `json:"compliance,omitempty"`
}

type OpenRouterModality struct {
	Type                  string                    `json:"type"`
	SupportedInputs       map[string]any            `json:"supported_inputs,omitempty"`
	SupportedParameters   map[string]any            `json:"supported_parameters,omitempty"`
	MaxLength             *OpenRouterLimit          `json:"max_length,omitempty"`
	Streaming             *bool                     `json:"streaming,omitempty"`
	Pricing               []OpenRouterPricingEntry  `json:"pricing,omitempty"`
	Capacity              []OpenRouterCapacityEntry `json:"capacity,omitempty"`
	PassthroughParameters map[string]any            `json:"passthrough_parameters,omitempty"`
}

type OpenRouterLimit struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

type OpenRouterPricingEntry struct {
	Type       string                    `json:"type"`
	Unit       string                    `json:"unit"`
	CostUSD    string                    `json:"cost_usd"`
	TTLSeconds *int64                    `json:"ttl_seconds,omitempty"`
	Implicit   bool                      `json:"implicit,omitempty"`
	Overrides  []OpenRouterPriceOverride `json:"overrides,omitempty"`
}

type OpenRouterPriceOverride struct {
	When    map[string]any `json:"when"`
	CostUSD string         `json:"cost_usd"`
}

type OpenRouterCapacityEntry struct {
	Type  string `json:"type"`
	Unit  string `json:"unit"`
	Per   string `json:"per,omitempty"`
	Value int64  `json:"value"`
}

type OpenRouterDatacenter struct {
	CountryCode string `json:"country_code"`
	Region      string `json:"region,omitempty"`
}

type OpenRouterCompliance struct {
	ZDR   bool `json:"zdr"`
	HIPAA bool `json:"hipaa"`
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
