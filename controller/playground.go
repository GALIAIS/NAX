package controller

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

func Playground(c *gin.Context) {
	if newAPIError := preparePlaygroundRelay(c, types.RelayFormatOpenAI); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	Relay(c, types.RelayFormatOpenAI)
}

// PlaygroundImage and PlaygroundTask share the dashboard-only playground
// context with chat. Keeping these endpoints under /pg is important: relay
// billing must use the authenticated user wallet/subscription and must not
// attempt to look up a synthetic API token.
func PlaygroundImage(c *gin.Context) {
	if newAPIError := preparePlaygroundRelay(c, types.RelayFormatOpenAIImage); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
}

func PlaygroundTask(c *gin.Context) {
	if newAPIError := preparePlaygroundRelay(c, types.RelayFormatTask); newAPIError != nil {
		respondPlaygroundError(c, newAPIError)
		return
	}
	RelayTask(c)
}

func preparePlaygroundRelay(c *gin.Context, relayFormat types.RelayFormat) *types.NewAPIError {
	if c.GetBool("use_access_token") {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, nil, nil)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	userCache, err := model.GetUserCache(c.GetInt("id"))
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	// Write user context to ensure acceptUnsetRatio and the current wallet
	// settings are available to pricing and settlement.
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: c.GetInt("id"),
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)
	return nil
}

func respondPlaygroundError(c *gin.Context, newAPIError *types.NewAPIError) {
	if newAPIError == nil {
		return
	}
	c.JSON(newAPIError.StatusCode, gin.H{
		"error": newAPIError.ToOpenAIError(),
	})
}
