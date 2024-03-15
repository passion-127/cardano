// Copyright 2023 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"time"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"github.com/blinklabs-io/cardano-node-api/internal/logging"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"

	_ "github.com/blinklabs-io/cardano-node-api/docs" // docs is generated by Swag CLI
	swaggerFiles "github.com/swaggo/files"            // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger"        // gin-swagger middleware
)

//	@title			cardano-node-api
//	@version		1.0
//	@description	Cardano Node API
//	@host			localhost
//	@Schemes		http
//	@BasePath		/api

//	@contact.name	Blink Labs
//	@contact.url	https://blinklabs.io
//	@contact.email	support@blinklabs.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html
func Start(cfg *config.Config) error {
	// Disable gin debug and color output
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	// Configure API router
	router := gin.New()
	// Catch panics and return a 500
	router.Use(gin.Recovery())
	// Standard logging
	logger := logging.GetLogger()
	// Access logging
	accessLogger := logging.GetAccessLogger()
	skipPaths := []string{}
	if cfg.Logging.Healthchecks {
		skipPaths = append(skipPaths, "/healthcheck")
		logger.Infof("disabling access logs for /healthcheck")
	}
	router.Use(ginzap.GinzapWithConfig(accessLogger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  skipPaths,
	}))
	router.Use(ginzap.RecoveryWithZap(accessLogger, true))

	// Create a healthcheck
	router.GET("/healthcheck", handleHealthcheck)
	// Create a swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Configure API routes
	apiGroup := router.Group("/api")
	configureLocalTxMonitorRoutes(apiGroup)
	configureLocalTxSubmissionRoutes(apiGroup)

	// Metrics
	metricsRouter := gin.New()
	metrics := ginmetrics.GetMonitor()
	// Set metrics path
	metrics.SetMetricPath("/")
	// Set metrics router
	metrics.Expose(metricsRouter)
	// Use metrics middleware without exposing path in main app router
	// We only collect metrics on the API endpoints
	metrics.UseWithoutExposingEndpoint(apiGroup)

	// Start metrics listener
	go func() {
		// TODO: return error if we cannot initialize metrics
		logger.Infof("starting metrics listener on %s:%d",
			cfg.Metrics.ListenAddress,
			cfg.Metrics.ListenPort)
		err := metricsRouter.Run(fmt.Sprintf("%s:%d",
			cfg.Metrics.ListenAddress,
			cfg.Metrics.ListenPort))
		if err != nil {
			logger.Fatalf("failed to start metrics listener: %s", err)
		}
	}()

	// Start API listener
	err := router.Run(fmt.Sprintf("%s:%d",
		cfg.Api.ListenAddress,
		cfg.Api.ListenPort))
	return err
}

type responseApiError struct {
	Msg string `json:"msg" example:"error message"`
}

func apiError(msg string) responseApiError {
	return responseApiError{
		Msg: msg,
	}
}

func handleHealthcheck(c *gin.Context) {
	// TODO: add some actual health checking here
	c.JSON(200, gin.H{"failed": false})
}
