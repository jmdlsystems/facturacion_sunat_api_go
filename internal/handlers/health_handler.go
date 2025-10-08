package handlers

import (
	"database/sql"
	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/pkg/sunat"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db         *sql.DB
	sunatClient *sunat.Client
	config     *config.Config
}

func NewHealthHandler(db *sql.DB, sunatClient *sunat.Client) *HealthHandler {
	return &HealthHandler{
		db:          db,
		sunatClient: sunatClient,
		config:      config.GetConfig(),
	}
}

func (h *HealthHandler) CheckHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Servicio funcionando correctamente",
		"version": "1.0.0",
	})
}

func (h *HealthHandler) CheckDatabase(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Base de datos no inicializada",
		})
		return
	}

	// Verificar conexión a la base de datos
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Error conectando a la base de datos",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Base de datos funcionando correctamente",
	})
}

func (h *HealthHandler) CheckSUNAT(c *gin.Context) {
	if h.sunatClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Cliente SUNAT no inicializado",
		})
		return
	}

	// Verificar conexión a SUNAT (ping básico)
	if err := h.sunatClient.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Error conectando a SUNAT",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Conexión a SUNAT funcionando correctamente",
	})
} 