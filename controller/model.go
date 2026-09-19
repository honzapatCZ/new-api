package controller

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel/ai360"
	"github.com/QuantumNous/new-api/relay/channel/lingyiwanwu"
	"github.com/QuantumNous/new-api/relay/channel/minimax"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

// https://platform.openai.com/docs/api-reference/models/list

var openAIModels []dto.OpenAIModels
var openAIModelsMap map[string]dto.OpenAIModels
var channelId2Models map[int][]string

func init() {
	// https://platform.openai.com/docs/models/model-endpoint-compatibility
	for i := range constant.APITypeDummy {
		if i == constant.APITypeAIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName()
		modelNames := adaptor.GetModelList()
		for _, modelName := range modelNames {
			openAIModels = append(openAIModels, dto.OpenAIModels{
				Id:      modelName,
				Object:  "model",
				Created: 1626777600,
				OwnedBy: channelName,
			})
		}
	}
	for _, modelName := range ai360.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: ai360.ChannelName,
		})
	}
	for _, modelName := range moonshot.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: moonshot.ChannelName,
		})
	}
	for _, modelName := range lingyiwanwu.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: lingyiwanwu.ChannelName,
		})
	}
	for _, modelName := range minimax.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: minimax.ChannelName,
		})
	}
	for modelName, _ := range constant.MidjourneyModel2Action {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: "midjourney",
		})
	}
	openAIModelsMap = make(map[string]dto.OpenAIModels)
	for _, aiModel := range openAIModels {
		openAIModelsMap[aiModel.Id] = aiModel
	}
	channelId2Models = make(map[int][]string)
	for i := 1; i <= constant.ChannelTypeDummy; i++ {
		apiType, success := common.ChannelType2APIType(i)
		if !success || apiType == constant.APITypeAIProxyLibrary {
			if plugin, ok := jsplugin.DefaultRegistry.GetByChannelType(i); ok {
				channelId2Models[i] = append([]string(nil), plugin.Meta.Models...)
			}
			continue
		}
		meta := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: i,
		}}
		adaptor := relay.GetAdaptor(apiType)
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
		if len(channelId2Models[i]) == 0 {
			if plugin, ok := jsplugin.DefaultRegistry.GetByChannelType(i); ok {
				channelId2Models[i] = append([]string(nil), plugin.Meta.Models...)
			}
		}
	}
	openAIModels = lo.UniqBy(openAIModels, func(m dto.OpenAIModels) string {
		return m.Id
	})
}

func channelOwnerName(channelType int) string {
	apiType, success := common.ChannelType2APIType(channelType)
	if !success {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType: channelType,
	}})
	if name := strings.TrimSpace(adaptor.GetChannelName()); name != "" {
		return name
	}
	return strings.ToLower(constant.GetChannelTypeName(channelType))
}

func getPreferredModelOwners(modelNames []string, groups []string) map[string]string {
	channelTypes, err := model.GetPreferredModelOwnerChannelTypes(modelNames, groups)
	if err != nil {
		common.SysLog(fmt.Sprintf("GetPreferredModelOwnerChannelTypes error: %v", err))
		return map[string]string{}
	}

	ownerByChannelType := make(map[int]string)
	owners := make(map[string]string, len(channelTypes))
	for modelName, channelType := range channelTypes {
		owner, ok := ownerByChannelType[channelType]
		if !ok {
			owner = channelOwnerName(channelType)
			ownerByChannelType[channelType] = owner
		}
		if owner != "" {
			owners[modelName] = owner
		}
	}
	return owners
}

func buildOpenAIModel(modelName string, ownerByModel map[string]string, pricingByModel map[string]model.Pricing) dto.OpenAIModels {
	var oaiModel dto.OpenAIModels
	if staticModel, ok := openAIModelsMap[modelName]; ok {
		oaiModel = staticModel
	} else {
		oaiModel = dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: "custom",
		}
	}
	if owner, ok := ownerByModel[modelName]; ok && owner != "" {
		oaiModel.OwnedBy = owner
	}
	oaiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(modelName)
	oaiModel.CanonicalSlug = modelName
	oaiModel.Name = modelName
	if pricing, ok := pricingByModel[modelName]; ok {
		if pricing.CreatedTime > 0 {
			oaiModel.Created = pricing.CreatedTime
		}
		oaiModel.Description = pricing.Description
		oaiModel.InputModalities, oaiModel.OutputModalities = openRouterModalities(pricing.SupportedEndpointTypes)
		oaiModel.Pricing = openRouterPricing(pricing)
	}
	return oaiModel
}

func openRouterModalities(endpointTypes []constant.EndpointType) ([]string, []string) {
	if len(endpointTypes) == 0 {
		return nil, nil
	}

	inputModalities := []string{"text"}
	outputModalities := []string{"text"}
	if slices.Contains(endpointTypes, constant.EndpointTypeImageGeneration) {
		inputModalities = append(inputModalities, "image")
		outputModalities = []string{"image"}
	} else if slices.Contains(endpointTypes, constant.EndpointTypeOpenAIVideo) {
		inputModalities = append(inputModalities, "image")
		outputModalities = []string{"video"}
	} else if slices.Contains(endpointTypes, constant.EndpointTypeEmbeddings) {
		outputModalities = []string{"embeddings"}
	}

	return inputModalities, outputModalities
}

func openRouterPricing(pricing model.Pricing) *dto.OpenRouterPricing {
	if pricing.QuotaType == 1 {
		return &dto.OpenRouterPricing{Prompt: "0", Completion: "0", Request: decimal.NewFromFloat(pricing.ModelPrice).String(), Image: "0", WebSearch: "0", InternalReasoning: "0", InputCacheRead: "0", InputCacheWrite: "0"}
	}
	if pricing.BillingMode == "tiered_expr" {
		if len(pricing.BillingUsageSchema) > 0 || strings.TrimSpace(pricing.BillingExpr) == "" {
			return nil
		}
		usedVars := billingexpr.UsedVars(pricing.BillingExpr)
		_, baseTrace, err := billingexpr.RunExpr(pricing.BillingExpr, billingexpr.TokenParams{})
		if err != nil {
			return nil
		}
		result := &dto.OpenRouterPricing{Prompt: "0", Completion: "0", Request: "0", Image: "0", WebSearch: "0", InternalReasoning: "0", InputCacheRead: "0", InputCacheWrite: "0"}
		if baseTrace.FixedPrice != nil && !usedVars["image_count"] {
			result.Request = decimal.NewFromFloat(*baseTrace.FixedPrice).String()
		}
		if usedVars["p"] {
			var ok bool
			result.Prompt, ok = openRouterExpressionTokenPrice(pricing.BillingExpr, billingexpr.TokenParams{Len: 1}, billingexpr.TokenParams{P: 1, Len: 1})
			if !ok {
				return nil
			}
		}
		if usedVars["c"] {
			var ok bool
			result.Completion, ok = openRouterExpressionTokenPrice(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.TokenParams{C: 1})
			if !ok {
				return nil
			}
		}
		if usedVars["cr"] {
			var ok bool
			result.InputCacheRead, ok = openRouterExpressionTokenPrice(pricing.BillingExpr, billingexpr.TokenParams{Len: 1}, billingexpr.TokenParams{CR: 1, Len: 1})
			if !ok {
				return nil
			}
		}
		if usedVars["cc"] {
			var ok bool
			result.InputCacheWrite, ok = openRouterExpressionTokenPrice(pricing.BillingExpr, billingexpr.TokenParams{Len: 1}, billingexpr.TokenParams{CC: 1, Len: 1})
			if !ok {
				return nil
			}
		} else if usedVars["cc1h"] {
			var ok bool
			result.InputCacheWrite, ok = openRouterExpressionTokenPrice(pricing.BillingExpr, billingexpr.TokenParams{Len: 1}, billingexpr.TokenParams{CC1h: 1, Len: 1})
			if !ok {
				return nil
			}
		}
		if usedVars["image_count"] {
			one := 1
			cost, _, runErr := billingexpr.RunExprWithRequest(pricing.BillingExpr, billingexpr.TokenParams{}, billingexpr.RequestInput{ImageCount: &one})
			if runErr != nil || cost < 0 {
				return nil
			}
			result.Image = decimal.NewFromFloat(cost).Div(decimal.NewFromInt(1_000_000)).String()
		}
		return result
	}
	if pricing.BillingMode != "" || common.QuotaPerUnit <= 0 {
		return nil
	}

	promptPrice := decimal.NewFromFloat(pricing.ModelRatio).Div(decimal.NewFromFloat(common.QuotaPerUnit))
	result := &dto.OpenRouterPricing{
		Prompt:            promptPrice.String(),
		Completion:        promptPrice.Mul(decimal.NewFromFloat(pricing.CompletionRatio)).String(),
		Request:           "0",
		Image:             "0",
		WebSearch:         "0",
		InternalReasoning: "0",
		InputCacheRead:    "0",
		InputCacheWrite:   "0",
	}
	if pricing.CacheRatio != nil {
		result.InputCacheRead = promptPrice.Mul(decimal.NewFromFloat(*pricing.CacheRatio)).String()
	}
	if pricing.CreateCacheRatio != nil {
		result.InputCacheWrite = promptPrice.Mul(decimal.NewFromFloat(*pricing.CreateCacheRatio)).String()
	}
	return result
}

func openRouterExpressionTokenPrice(expression string, baseline, priced billingexpr.TokenParams) (string, bool) {
	baseCost, _, baseErr := billingexpr.RunExpr(expression, baseline)
	cost, trace, runErr := billingexpr.RunExpr(expression, priced)
	if baseErr != nil || runErr != nil || trace.BillingUnit != billingexpr.BillingUnitToken || cost < baseCost {
		return "", false
	}
	return decimal.NewFromFloat(cost - baseCost).Div(decimal.NewFromInt(1_000_000)).String(), true
}

type modelListGroups struct {
	userGroup   string
	tokenGroup  string
	ownerGroups []string
}

func getModelListGroups(c *gin.Context) (modelListGroups, error) {
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup == "" && (tokenGroup == "" || tokenGroup == "auto") {
		var err error
		userGroup, err = model.GetUserGroup(c.GetInt("id"), false)
		if err != nil {
			return modelListGroups{}, err
		}
	}

	if tokenGroup == "auto" {
		return modelListGroups{
			userGroup:   userGroup,
			tokenGroup:  tokenGroup,
			ownerGroups: service.GetRequestAutoGroups(c, userGroup),
		}, nil
	}

	group := userGroup
	if tokenGroup != "" {
		group = tokenGroup
	}
	return modelListGroups{
		userGroup:   userGroup,
		tokenGroup:  tokenGroup,
		ownerGroups: []string{group},
	}, nil
}

func ListModels(c *gin.Context, modelType int) {
	acceptUnsetRatioModel := operation_setting.SelfUseModeEnabled
	if !acceptUnsetRatioModel {
		userId := c.GetInt("id")
		if userId > 0 {
			userSettings, _ := model.GetUserSetting(userId, false)
			if userSettings.AcceptUnsetRatioModel {
				acceptUnsetRatioModel = true
			}
		}
	}

	userModelNames := make([]string, 0)
	groups, err := getModelListGroups(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "get user group failed",
		})
		return
	}
	ownerGroups := groups.ownerGroups
	modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	var tokenModelLimit map[string]bool
	if modelLimitEnable {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		if ok {
			tokenModelLimit, _ = s.(map[string]bool)
		}
		if tokenModelLimit == nil {
			tokenModelLimit = map[string]bool{}
		}
	}
	models := service.GetGroupsEnabledModels(ownerGroups)
	for _, modelName := range models {
		if modelLimitEnable {
			matchingName := ratio_setting.RoutingMatchModelName(modelName)
			if !tokenModelLimit[modelName] && !tokenModelLimit[matchingName] {
				continue
			}
		}
		if !acceptUnsetRatioModel && !helper.HasModelBillingConfig(modelName) {
			continue
		}
		userModelNames = append(userModelNames, modelName)
	}

	ownerByModel := map[string]string{}
	if len(ownerGroups) > 0 {
		ownerByModel = getPreferredModelOwners(userModelNames, ownerGroups)
	}
	pricingByModel := make(map[string]model.Pricing, len(userModelNames))
	for _, pricing := range model.GetPricing() {
		pricingByModel[pricing.ModelName] = pricing
	}
	userOpenAiModels := make([]dto.OpenAIModels, 0, len(userModelNames))
	for _, modelName := range userModelNames {
		userOpenAiModels = append(userOpenAiModels, buildOpenAIModel(modelName, ownerByModel, pricingByModel))
	}

	switch modelType {
	case constant.ChannelTypeAnthropic:
		useranthropicModels := make([]dto.AnthropicModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			useranthropicModels[i] = dto.AnthropicModel{
				ID:          model.Id,
				CreatedAt:   time.Unix(model.Created, 0).UTC().Format(time.RFC3339),
				DisplayName: model.Id,
				Type:        "model",
			}
		}
		firstID := ""
		lastID := ""
		if len(useranthropicModels) > 0 {
			firstID = useranthropicModels[0].ID
			lastID = useranthropicModels[len(useranthropicModels)-1].ID
		}
		c.JSON(200, gin.H{
			"data":     useranthropicModels,
			"first_id": firstID,
			"has_more": false,
			"last_id":  lastID,
		})
	case constant.ChannelTypeGemini:
		userGeminiModels := make([]dto.GeminiModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			userGeminiModels[i] = dto.GeminiModel{
				Name:        model.Id,
				DisplayName: model.Id,
			}
		}
		c.JSON(200, gin.H{
			"models":        userGeminiModels,
			"nextPageToken": nil,
		})
	default:
		c.JSON(200, gin.H{
			"success": true,
			"data":    userOpenAiModels,
			"object":  "list",
		})
	}
}

func ChannelListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    openAIModels,
	})
}

func DashboardListModels(c *gin.Context) {
	modelsByChannel := make(map[int][]string, len(channelId2Models))
	for channelType, models := range channelId2Models {
		modelsByChannel[channelType] = append([]string(nil), models...)
	}
	for channelType := 1; channelType <= constant.ChannelTypeDummy; channelType++ {
		if plugin, ok := jsplugin.DefaultRegistry.GetByChannelType(channelType); ok {
			modelsByChannel[channelType] = append([]string(nil), plugin.Meta.Models...)
		}
	}
	c.JSON(200, gin.H{
		"success": true,
		"data":    modelsByChannel,
	})
}

func EnabledListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    model.GetEnabledModels(),
	})
}

func RetrieveModel(c *gin.Context, modelType int) {
	modelId := c.Param("model")
	if aiModel, ok := openAIModelsMap[modelId]; ok {
		switch modelType {
		case constant.ChannelTypeAnthropic:
			c.JSON(200, dto.AnthropicModel{
				ID:          aiModel.Id,
				CreatedAt:   time.Unix(aiModel.Created, 0).UTC().Format(time.RFC3339),
				DisplayName: aiModel.Id,
				Type:        "model",
			})
		default:
			c.JSON(200, aiModel)
		}
	} else {
		openAIError := types.OpenAIError{
			Message: fmt.Sprintf("The model '%s' does not exist", modelId),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
	}
}
