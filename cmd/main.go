package main

import (
	"github.com/gin-gonic/gin"
	"passwordCrakerBackend/internal/core/service"
	"passwordCrakerBackend/internal/platform/desktop"
	"passwordCrakerBackend/internal/platform/mobile"
	"passwordCrakerBackend/internal/platform/web"
)

func main() {
	// Initialize core service
	crackingService := service.NewCrackingService(repo, hashService)

	// Setup web platform
	router := gin.Default()
	webHandler := web.NewWebHandler(crackingService)
	web.SetupRoutes(router, webHandler)

	// Setup mobile platform
	mobileBinding := mobile.NewMobileBinding(crackingService)

	// Setup desktop platform
	desktopConfig := desktop.NewDefaultConfig()
	desktopLib := desktop.NewDesktopLib(crackingService, desktopConfig)

	// Start web server
	router.Run(":8080")
}
