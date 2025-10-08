package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// CORS middleware para permitir requests desde diferentes orígenes
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Lista de orígenes permitidos
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}
		
		// Verificar si el origen está permitido
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}
		
		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Content-Disposition")
		c.Header("Access-Control-Max-Age", "86400")
		
		// Manejar preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		c.Next()
	}
}

// RequestLogger middleware para logging de requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log request details si es necesario para debugging
		// method := c.Request.Method
		// path := c.Request.URL.Path
		
		c.Next()
		
		// Log response details
		status := c.Writer.Status()
		
		// Solo log errores para requests importantes
		if status >= 400 {
			gin.Logger()(c)
		}
	}
}

// ErrorHandler middleware para manejo centralizado de errores
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		// Verificar si hay errores
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Respuesta de error estandarizada
			errorResponse := gin.H{
				"error":   true,
				"message": err.Error(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
			}
			
			// Determinar código de status basado en el tipo de error
			status := http.StatusInternalServerError
			if c.Writer.Status() != http.StatusOK {
				status = c.Writer.Status()
			}
			
			c.JSON(status, errorResponse)
		}
	}
}

// Security middleware para headers de seguridad básicos
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		c.Next()
	}
}