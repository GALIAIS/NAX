package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetVideoRouterRegistersRoutesWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	require.NotPanics(t, func() {
		SetVideoRouter(engine)
	})
}

func TestSetVideoRouterRegistersPlaygroundMediaRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	routes := engine.Routes()
	hasRoute := func(method string, path string) bool {
		for _, route := range routes {
			if route.Method == method && route.Path == path {
				return true
			}
		}
		return false
	}

	require.True(t, hasRoute("POST", "/pg/videos"))
	require.True(t, hasRoute("GET", "/pg/videos/:task_id"))
	require.True(t, hasRoute("GET", "/pg/videos/:task_id/content"))
	require.True(t, hasRoute("DELETE", "/pg/videos/:task_id"))
}
