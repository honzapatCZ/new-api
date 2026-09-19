package dto

import (
	"github.com/QuantumNous/new-api/relaykit/types"
)

// OpenAIModels is the internal model-list value. Its JSON representation is
// emitted as an OpenRouter provider-monitor 2.4 document.
type OpenAIModels struct {
	Id                     string               `json:"id"`
	Object                 string               `json:"object"`
	Created                int64                `json:"created"`
	OwnedBy                string               `json:"owned_by"`
	SupportedEndpointTypes []types.EndpointType `json:"supported_endpoint_types"`
	CanonicalSlug          string               `json:"canonical_slug,omitempty"`
	Name                   string               `json:"name,omitempty"`
	Description            string               `json:"description,omitempty"`
	InputModalities        any                  `json:"input_modalities,omitempty"`
	OutputModalities       any                  `json:"output_modalities,omitempty"`
	Pricing                any                  `json:"pricing,omitempty"`
	PricingDetails         *OpenRouterPricingDetails `json:"pricing_details,omitempty"`
}

// OpenRouterModality is a typed 2.4 input/output modality declaration.
type OpenRouterModality struct {
	Type               string                       `json:"type"`
	SupportedInputs   map[string]any               `json:"supported_inputs,omitempty"`
	SupportedParameters map[string]any              `json:"supported_parameters,omitempty"`
	MaxLength          *OpenRouterLimit              `json:"max_length,omitempty"`
	Streaming          *bool                        `json:"streaming,omitempty"`
	Pricing            []OpenRouterPricingEntry      `json:"pricing,omitempty"`
	Capacity           []OpenRouterCapacityEntry     `json:"capacity,omitempty"`
	PassthroughParameters map[string]any            `json:"passthrough_parameters,omitempty"`
}

type OpenRouterLimit struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

type OpenRouterPricingEntry struct {
	Type       string                     `json:"type"`
	Unit       string                     `json:"unit"`
	CostUSD    string                     `json:"cost_usd"`
	TTLSeconds *int64                    `json:"ttl_seconds,omitempty"`
	Implicit   bool                       `json:"implicit,omitempty"`
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

// OpenRouterPricingDetails is an additive New API field used for expressions
// that cannot be losslessly represented by static 2.4 pricing entries. The
// Model Gallery consumes the same expression and usage-schema information.
type OpenRouterPricingDetails struct {
	Mode       string                             `json:"mode"`
	Expression string                             `json:"expression,omitempty"`
	Usage      map[string]OpenRouterUsageField    `json:"usage,omitempty"`
	Providers  []OpenRouterProviderPricingVariant `json:"providers,omitempty"`
}

type OpenRouterUsageField struct {
	Type string `json:"type,omitempty"`
	Unit string `json:"unit,omitempty"`
}

type OpenRouterProviderPricingVariant struct {
	Key        string                          `json:"key"`
	Name       string                          `json:"name,omitempty"`
	Mode       string                          `json:"mode,omitempty"`
	Expression string                          `json:"expression,omitempty"`
	Usage      map[string]OpenRouterUsageField `json:"usage,omitempty"`
}

type OpenRouterPricing struct {
	Prompt            string `json:"prompt"`
	Completion        string `json:"completion"`
	Request           string `json:"request"`
	Image             string `json:"image"`
	WebSearch         string `json:"web_search"`
	InternalReasoning string `json:"internal_reasoning"`
	InputCacheRead    string `json:"input_cache_read"`
	InputCacheWrite   string `json:"input_cache_write"`
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
