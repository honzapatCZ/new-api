package dto

import "github.com/QuantumNous/new-api/relaykit/types"

// OpenAIModels is the common model-list representation. The additional fields
// are compatible with OpenRouter's model metadata response.
type OpenAIModels struct {
	Id                     string               `json:"id"`
	Object                 string               `json:"object"`
	Created                int64                `json:"created"`
	OwnedBy                string               `json:"owned_by"`
	SupportedEndpointTypes []types.EndpointType `json:"supported_endpoint_types"`
	CanonicalSlug          string               `json:"canonical_slug,omitempty"`
	Name                   string               `json:"name,omitempty"`
	Description            string               `json:"description,omitempty"`
	InputModalities        []string             `json:"input_modalities,omitempty"`
	OutputModalities       []string             `json:"output_modalities,omitempty"`
	Pricing                *OpenRouterPricing   `json:"pricing,omitempty"`
	PricingDetails         *OpenRouterPricingDetails `json:"pricing_details,omitempty"`
}

// OpenRouterPricing follows the pricing keys exposed by OpenRouter.
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

// PricingDetails preserves New API billing information that cannot be safely
// projected into OpenRouter's flat pricing object, such as task expressions.
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
