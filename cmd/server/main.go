package main

import (
	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/internal/handlers"
	"facturacion_sunat_api_go/internal/middleware"
	"facturacion_sunat_api_go/internal/repository"
	"facturacion_sunat_api_go/internal/services"
	"facturacion_sunat_api_go/pkg/certificate"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando configuración: %v", err)
	}

	// Inicializar base de datos
	db, err := repository.InitDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Error inicializando base de datos: %v", err)
	}
	defer db.Close()

	// Inicializar repositorios
	comprobanteRepo := repository.NewComprobanteRepository(db)

	// Inicializar servicios
	certManager := certificate.NewManager()
	ublService := services.NewUBLService()
	conversionService := services.NewConversionService(ublService)
	signingService := services.NewSigningService(certManager, ublService)
	encodingService := services.NewEncodingService()
	sunatService := services.NewSUNATService(&cfg.SUNAT, encodingService)

	// Inicializar handlers
	healthHandler := handlers.NewHealthHandler(db, sunatService.Client)
	comprobanteHandler := handlers.NewComprobanteHandler(
		comprobanteRepo,
		conversionService,
		signingService,
		encodingService,
		sunatService,
	)

	// Configurar router
	router := setupRouter(healthHandler, comprobanteHandler)

	// Configurar servidor
	server := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: router,
	}

	log.Printf("Servidor iniciando en %s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Entorno: %s", getEnvironment())
	log.Printf("Base de datos: %s", cfg.Database.Database)

	// Iniciar servidor
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}

func setupRouter(healthHandler *handlers.HealthHandler, comprobanteHandler *handlers.ComprobanteHandler) *gin.Engine {
	// Configurar modo Gin
	if getEnvironment() == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware global
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// Rutas de salud
	health := router.Group("/health")
	{
		health.GET("/", healthHandler.CheckHealth)
		health.GET("/database", healthHandler.CheckDatabase)
		health.GET("/sunat", healthHandler.CheckSUNAT)
	}

	// API v1 - APIs principales + consultas
	v1 := router.Group("/api/v1")
	{
		// Rutas RESTful estándar para facturación electrónica
		v1.POST("/invoices", comprobanteHandler.CreateFactura)
		v1.POST("/credit-notes", comprobanteHandler.CreateNotaCredito)
		v1.POST("/debit-notes", comprobanteHandler.CreateNotaDebito)
		v1.GET("/documents/:id/status", comprobanteHandler.GetComprobanteResult)
		v1.GET("/documents/:id/xml", comprobanteHandler.DownloadXML)
		v1.GET("/documents/:id/pdf", comprobanteHandler.DownloadPDF)

		// Rutas legacy/compatibilidad
		comprobantes := v1.Group("/comprobantes")
		{
			comprobantes.POST("/", comprobanteHandler.CreateComprobante)
			comprobantes.POST("/generate-xml", comprobanteHandler.GenerateXMLWithoutSignature)
			comprobantes.GET("/", comprobanteHandler.ListComprobantes)
			comprobantes.GET("/:id", comprobanteHandler.GetComprobante)
			comprobantes.POST("/:id/sign", comprobanteHandler.SignComprobante)
			comprobantes.POST("/:id/send", comprobanteHandler.SendToSUNAT)
			comprobantes.GET("/:id/result", comprobanteHandler.GetComprobanteResult)
		}

		utils := v1.Group("/utils")
		{
			utils.POST("/convert-ubl", comprobanteHandler.ConvertToUBL)
			utils.POST("/calculate-totals", comprobanteHandler.CalculateTotals)
		}
	}

	// Endpoints requeridos por API FE Perú (fuera de /api/v1 para cumplir con el estándar del PDF)
	fe := router.Group("/")
	fe.Use(middleware.APIKeyAuth())
	{
		fe.GET("api/contribuyente", comprobanteHandler.GetContribuyente)
		fe.GET("api/validar_ruc", comprobanteHandler.ValidateRUC)
		fe.GET("api/soles", comprobanteHandler.GetTasaCambio)
		fe.GET("api/calculadora", comprobanteHandler.CalculadoraCambio)
	}

	return router
}

func getEnvironment() string {
	env := os.Getenv("GO_ENV")
	if env == "" {
		return "development"
	}
	return env
}