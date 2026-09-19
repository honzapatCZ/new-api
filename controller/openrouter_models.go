package controller

import (
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/shopspring/decimal"
)

func openRouterModelDocument(pricing model.Pricing) ([]dto.OpenRouterInputModality, []dto.OpenRouterOutputModality, []dto.OpenRouterPrice) {
	outputType := "text"
	streaming := true
	plugin, hasPlugin := jsplugin.DefaultRegistry.Generation().GetByModel(pricing.ModelName)
	switch {
	case hasPlugin && slices.ContainsFunc(plugin.Meta.Protocols, func(claim jsplugin.ProtocolClaim) bool {
		return claim.Name == "openai_video" && (len(claim.Models) == 0 || slices.Contains(claim.Models, pricing.ModelName))
	}):
		outputType, streaming = "video", false
	case slices.Contains(pricing.SupportedEndpointTypes, constant.EndpointTypeImageGeneration):
		outputType, streaming = "image", false
	case slices.Contains(pricing.SupportedEndpointTypes, constant.EndpointTypeOpenAIVideo):
		outputType, streaming = "video", false
	case slices.Contains(pricing.SupportedEndpointTypes, constant.EndpointTypeEmbeddings):
		outputType, streaming = "embeddings", false
	case slices.Contains(pricing.SupportedEndpointTypes, constant.EndpointTypeJinaRerank):
		outputType, streaming = "rerank", false
	case strings.Contains(strings.ToLower(pricing.ModelName), "transcri") || strings.Contains(strings.ToLower(pricing.ModelName), "whisper"):
		outputType = "transcription"
	case strings.Contains(strings.ToLower(pricing.ModelName), "tts") || strings.Contains(strings.ToLower(pricing.ModelName), "speech"):
		outputType = "speech"
	case strings.Contains(strings.ToLower(pricing.ModelName), "suno") || strings.Contains(strings.ToLower(pricing.ModelName), "audio"):
		outputType = "audio"
	}

	inputs := []dto.OpenRouterInputModality{{Type: "text"}}
	if outputType == "image" || outputType == "video" {
		inputs = append(inputs, dto.OpenRouterInputModality{Type: "image"})
	} else if outputType == "transcription" {
		inputs = append(inputs, dto.OpenRouterInputModality{Type: "audio"})
	}
	parameters := make(map[string]dto.OpenRouterCapability)
	for _, name := range slices.Sorted(maps.Keys(pricing.BillingUsageSchema)) {
		field := pricing.BillingUsageSchema[name]
		if len(field.Enum) > 0 {
			values := make([]any, len(field.Enum))
			for i := range field.Enum {
				values[i] = field.Enum[i]
			}
			parameters[name] = dto.OpenRouterCapability{Type: "enum", Values: values}
		} else if field.Type == "boolean" {
			parameters[name] = dto.OpenRouterCapability{Type: "boolean"}
		} else if field.Type == "number" {
			minimum := float64(0)
			maximum := float64(math.MaxInt32)
			switch field.Unit {
			case "second":
				maximum = float64(relaycommon.MaxTaskDurationSeconds)
			case "count":
				maximum = float64(dto.MaxImageN)
			}
			parameters[name] = dto.OpenRouterCapability{Type: "integer", Min: &minimum, Max: &maximum, Unit: field.Unit}
		}
	}
	output := dto.OpenRouterOutputModality{Type: outputType, SupportedParameters: parameters}
	if outputType != "embeddings" && outputType != "rerank" {
		output.Streaming = &streaming
	}

	rootPrices := make([]dto.OpenRouterPrice, 0)
	if pricing.QuotaType == 1 {
		price := dto.OpenRouterPrice{Type: "request", Unit: "request", CostUSD: decimal.NewFromFloat(pricing.ModelPrice).String()}
		if outputType == "image" {
			price.Type, price.Unit = "completion", "image"
			output.Pricing = append(output.Pricing, price)
		} else {
			rootPrices = append(rootPrices, price)
		}
		return inputs, []dto.OpenRouterOutputModality{output}, rootPrices
	}

	if pricing.BillingMode == "tiered_expr" && len(pricing.BillingUsageSchema) > 0 {
		output.Pricing, rootPrices = openRouterTaskPrices(pricing, outputType)
		return inputs, []dto.OpenRouterOutputModality{output}, rootPrices
	}

	prices := openRouterTokenPrices(pricing)
	outputs := []dto.OpenRouterOutputModality{output}
	for _, price := range prices {
		switch price.Type {
		case "prompt", "cached_prompt", "cache_write":
			inputs[0].Pricing = append(inputs[0].Pricing, price)
		case "image_prompt":
			price.Type = "prompt"
			price.Unit = "token"
			if len(inputs) == 1 {
				inputs = append(inputs, dto.OpenRouterInputModality{Type: "image"})
			}
			inputs[1].Pricing = append(inputs[1].Pricing, price)
		case "audio_prompt":
			price.Type = "prompt"
			price.Unit = "token"
			inputs = append(inputs, dto.OpenRouterInputModality{Type: "audio", Pricing: []dto.OpenRouterPrice{price}})
		case "request":
			rootPrices = append(rootPrices, price)
		case "image_completion", "audio_completion":
			modality := strings.TrimSuffix(price.Type, "_completion")
			price.Type = "completion"
			index := slices.IndexFunc(outputs, func(item dto.OpenRouterOutputModality) bool { return item.Type == modality })
			if index < 0 {
				streaming := modality == "audio"
				outputs = append(outputs, dto.OpenRouterOutputModality{Type: modality, Streaming: &streaming, SupportedParameters: map[string]dto.OpenRouterCapability{}})
				index = len(outputs) - 1
			}
			outputs[index].Pricing = append(outputs[index].Pricing, price)
		default:
			outputs[0].Pricing = append(outputs[0].Pricing, price)
		}
	}
	return inputs, outputs, rootPrices
}

func openRouterTokenPrices(pricing model.Pricing) []dto.OpenRouterPrice {
	if pricing.BillingMode == "tiered_expr" {
		return openRouterExpressionPrices(pricing.BillingExpr)
	}
	if pricing.BillingMode != "" || common.QuotaPerUnit <= 0 {
		return nil
	}
	base := decimal.NewFromFloat(pricing.ModelRatio).Div(decimal.NewFromFloat(common.QuotaPerUnit))
	prices := []dto.OpenRouterPrice{
		{Type: "prompt", Unit: "token", CostUSD: base.String()},
		{Type: "completion", Unit: "token", CostUSD: base.Mul(decimal.NewFromFloat(pricing.CompletionRatio)).String()},
	}
	if pricing.CacheRatio != nil {
		prices = append(prices, dto.OpenRouterPrice{Type: "cached_prompt", Unit: "token", CostUSD: base.Mul(decimal.NewFromFloat(*pricing.CacheRatio)).String()})
	}
	if pricing.CreateCacheRatio != nil {
		prices = append(prices, dto.OpenRouterPrice{Type: "cache_write", Unit: "token", CostUSD: base.Mul(decimal.NewFromFloat(*pricing.CreateCacheRatio)).String()})
	}
	if pricing.ImageRatio != nil {
		prices = append(prices, dto.OpenRouterPrice{Type: "image_prompt", Unit: "token", CostUSD: base.Mul(decimal.NewFromFloat(*pricing.ImageRatio)).String()})
	}
	if pricing.AudioRatio != nil {
		audio := base.Mul(decimal.NewFromFloat(*pricing.AudioRatio))
		prices = append(prices, dto.OpenRouterPrice{Type: "audio_prompt", Unit: "token", CostUSD: audio.String()})
		if pricing.AudioCompletionRatio != nil {
			prices = append(prices, dto.OpenRouterPrice{Type: "audio_completion", Unit: "token", CostUSD: audio.Mul(decimal.NewFromFloat(*pricing.AudioCompletionRatio)).String()})
		}
	}
	return prices
}

func openRouterExpressionPrices(expression string) []dto.OpenRouterPrice {
	if strings.TrimSpace(expression) == "" {
		return nil
	}
	used := billingexpr.UsedVars(expression)
	for _, unsupported := range []string{"hour", "minute", "weekday", "month", "day", "param", "header"} {
		if used[unsupported] {
			return nil
		}
	}
	definitions := []struct {
		variable, priceType, unit string
		baseline, priced          billingexpr.TokenParams
		ttl                       int
	}{
		{variable: "p", priceType: "prompt", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{P: 1, Len: 1}},
		{variable: "c", priceType: "completion", unit: "token", priced: billingexpr.TokenParams{C: 1}},
		{variable: "cr", priceType: "cached_prompt", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{CR: 1, Len: 1}},
		{variable: "cc", priceType: "cache_write", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{CC: 1, Len: 1}, ttl: 300},
		{variable: "cc1h", priceType: "cache_write", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{CC1h: 1, Len: 1}, ttl: 3600},
		{variable: "img", priceType: "image_prompt", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{Img: 1, Len: 1}},
		{variable: "img_o", priceType: "image_completion", unit: "token", priced: billingexpr.TokenParams{ImgO: 1}},
		{variable: "ai", priceType: "audio_prompt", unit: "token", baseline: billingexpr.TokenParams{Len: 1}, priced: billingexpr.TokenParams{AI: 1, Len: 1}},
		{variable: "ao", priceType: "audio_completion", unit: "token", priced: billingexpr.TokenParams{AO: 1}},
	}
	prices := make([]dto.OpenRouterPrice, 0, len(definitions))
	priceVariables := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		if !used[definition.variable] {
			continue
		}
		value, ok := openRouterExpressionTokenPrice(expression, definition.baseline, definition.priced)
		if !ok {
			return nil
		}
		price := dto.OpenRouterPrice{Type: definition.priceType, Unit: definition.unit, CostUSD: value}
		if definition.ttl > 0 {
			price.TTLSeconds = &definition.ttl
		}
		prices = append(prices, price)
		priceVariables = append(priceVariables, definition.variable)
	}
	contextPattern := regexp.MustCompile(`\b(len|p)\s*(<=|<)\s*(\d+)`)
	for _, match := range contextPattern.FindAllStringSubmatch(expression, -1) {
		threshold, err := strconv.ParseInt(match[3], 10, 53)
		if err != nil {
			continue
		}
		if match[2] == "<=" {
			threshold++
		}
		for index, variable := range priceVariables {
			value, ok := openRouterExpressionPriceAtContext(expression, variable, float64(threshold))
			if ok && value != prices[index].CostUSD {
				prices[index].Overrides = append(prices[index].Overrides, dto.OpenRouterPriceOverride{
					When: map[string]any{"prompt_tokens": map[string]any{"gte": threshold}}, CostUSD: value,
				})
			}
		}
	}
	if used["image_count"] {
		one := 1
		cost, _, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{}, billingexpr.RequestInput{ImageCount: &one})
		if err != nil || cost < 0 {
			return nil
		}
		prices = append(prices, dto.OpenRouterPrice{Type: "completion", Unit: "image", CostUSD: decimal.NewFromFloat(cost).Div(decimal.NewFromInt(1_000_000)).String()})
	} else if _, trace, err := billingexpr.RunExpr(expression, billingexpr.TokenParams{}); err == nil && trace.FixedPrice != nil {
		prices = append(prices, dto.OpenRouterPrice{Type: "request", Unit: "request", CostUSD: decimal.NewFromFloat(*trace.FixedPrice).String()})
	}
	return prices
}

func openRouterExpressionPriceAtContext(expression, variable string, contextLength float64) (string, bool) {
	baseline := billingexpr.TokenParams{Len: contextLength}
	priced := baseline
	switch variable {
	case "p":
		baseline.P, priced.P = contextLength, contextLength+1
	case "c":
		priced.C = 1
	case "cr":
		priced.CR = 1
	case "cc":
		priced.CC = 1
	case "cc1h":
		priced.CC1h = 1
	case "img":
		priced.Img = 1
	case "img_o":
		priced.ImgO = 1
	case "ai":
		priced.AI = 1
	case "ao":
		priced.AO = 1
	default:
		return "", false
	}
	return openRouterExpressionTokenPrice(expression, baseline, priced)
}

func openRouterTaskPrices(pricing model.Pricing, outputType string) ([]dto.OpenRouterPrice, []dto.OpenRouterPrice) {
	usage := make(map[string]any)
	for _, name := range slices.Sorted(maps.Keys(pricing.BillingUsageSchema)) {
		field := pricing.BillingUsageSchema[name]
		switch {
		case len(field.Enum) > 0:
			usage[name] = field.Enum[0]
		case field.Type == "boolean":
			usage[name] = false
		case field.Type == "number":
			usage[name] = float64(0)
		}
	}
	baseCost, _, err := billingexpr.RunExprWithRequest(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: usage})
	if err != nil || baseCost < 0 {
		return nil, nil
	}
	outputPrices := make([]dto.OpenRouterPrice, 0)
	numericFields := make([]string, 0)
	for _, name := range slices.Sorted(maps.Keys(pricing.BillingUsageSchema)) {
		field := pricing.BillingUsageSchema[name]
		if field.Type != "number" {
			continue
		}
		pricedUsage := make(map[string]any, len(usage))
		maps.Copy(pricedUsage, usage)
		pricedUsage[name] = float64(1)
		cost, _, runErr := billingexpr.RunExprWithRequest(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: pricedUsage})
		if runErr != nil || cost < baseCost {
			return nil, nil
		}
		unit := field.Unit
		if unit == "count" && outputType == "image" {
			unit = "image"
		} else if unit == "count" || unit == "credit" {
			unit = "request"
		}
		if !slices.Contains([]string{"token", "image", "megapixel", "second", "character", "request"}, unit) {
			continue
		}
		numericFields = append(numericFields, name)
		outputPrices = append(outputPrices, dto.OpenRouterPrice{Type: "completion", Unit: unit, CostUSD: decimal.NewFromFloat(cost - baseCost).String()})
	}
	var rootPrices []dto.OpenRouterPrice
	if baseCost > 0 {
		rootPrices = append(rootPrices, dto.OpenRouterPrice{Type: "request", Unit: "request", CostUSD: decimal.NewFromFloat(baseCost).String()})
	}
	for _, candidate := range openRouterTaskUsageCombinations(pricing.BillingUsageSchema) {
		if reflect.DeepEqual(candidate, usage) {
			continue
		}
		candidateBaseCost, _, runErr := billingexpr.RunExprWithRequest(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: candidate})
		if runErr != nil || candidateBaseCost < 0 {
			return nil, nil
		}
		when := make(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(pricing.BillingUsageSchema)) {
			field := pricing.BillingUsageSchema[name]
			if len(field.Enum) > 0 || field.Type == "boolean" {
				when[name] = map[string]any{"equals": candidate[name]}
			}
		}
		for index, name := range numericFields {
			pricedUsage := make(map[string]any, len(candidate))
			maps.Copy(pricedUsage, candidate)
			pricedUsage[name] = float64(1)
			cost, _, priceErr := billingexpr.RunExprWithRequest(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: pricedUsage})
			if priceErr != nil || cost < candidateBaseCost {
				return nil, nil
			}
			value := decimal.NewFromFloat(cost - candidateBaseCost).String()
			if value != outputPrices[index].CostUSD {
				outputPrices[index].Overrides = append(outputPrices[index].Overrides, dto.OpenRouterPriceOverride{When: when, CostUSD: value})
			}
		}
		if candidateBaseCost > 0 && len(rootPrices) > 0 {
			value := decimal.NewFromFloat(candidateBaseCost).String()
			if value != rootPrices[0].CostUSD {
				rootPrices[0].Overrides = append(rootPrices[0].Overrides, dto.OpenRouterPriceOverride{When: when, CostUSD: value})
			}
		}
	}
	return outputPrices, rootPrices
}

func openRouterTaskUsageCombinations(schema map[string]jsplugin.UsageFieldSchema) []map[string]any {
	combinations := []map[string]any{{}}
	for _, name := range slices.Sorted(maps.Keys(schema)) {
		field := schema[name]
		values := make([]any, 0, len(field.Enum))
		for _, value := range field.Enum {
			values = append(values, value)
		}
		if field.Type == "boolean" {
			values = []any{false, true}
		}
		if len(values) == 0 {
			continue
		}
		next := make([]map[string]any, 0, len(combinations)*len(values))
		for _, combination := range combinations {
			for _, value := range values {
				item := make(map[string]any, len(combination)+1)
				maps.Copy(item, combination)
				item[name] = value
				next = append(next, item)
			}
		}
		combinations = next
	}
	for _, combination := range combinations {
		for _, name := range slices.Sorted(maps.Keys(schema)) {
			field := schema[name]
			if _, exists := combination[name]; exists {
				continue
			}
			switch {
			case field.Type == "boolean":
				combination[name] = false
			case field.Type == "number":
				combination[name] = float64(0)
			}
		}
	}
	return combinations
}

func openRouterExpressionTokenPrice(expression string, baseline, priced billingexpr.TokenParams) (string, bool) {
	baseCost, _, baseErr := billingexpr.RunExpr(expression, baseline)
	cost, trace, runErr := billingexpr.RunExpr(expression, priced)
	if baseErr != nil || runErr != nil || trace.BillingUnit != billingexpr.BillingUnitToken || cost < baseCost {
		return "", false
	}
	return decimal.NewFromFloat(cost - baseCost).Div(decimal.NewFromInt(1_000_000)).String(), true
}
