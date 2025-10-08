package main

import (
	"fmt"
	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/pkg/sunat"
)

func main() {
	fmt.Println("🔍 Verificando configuración SUNAT...")
	
	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Error cargando configuración: %v\n", err)
		return
	}
	
	fmt.Println("✅ Configuración cargada exitosamente")
	
	// Mostrar configuración SUNAT
	fmt.Printf("\n📋 Configuración SUNAT:\n")
	fmt.Printf("   BaseURL: %s\n", cfg.SUNAT.BaseURL)
	fmt.Printf("   BetaURL: %s\n", cfg.SUNAT.BetaURL)
	fmt.Printf("   FEBeta: %s\n", cfg.SUNAT.FEBeta)
	fmt.Printf("   RUC: %s\n", cfg.SUNAT.RUC)
	fmt.Printf("   Username: %s\n", cfg.SUNAT.Username)
	fmt.Printf("   Password: %s\n", cfg.SUNAT.Password)
	fmt.Printf("   Timeout: %d\n", cfg.SUNAT.Timeout)
	fmt.Printf("   MaxRetries: %d\n", cfg.SUNAT.MaxRetries)
	
	// Crear cliente SUNAT
	client := sunat.NewClient(&cfg.SUNAT)
	
	fmt.Printf("\n🔗 Cliente SUNAT creado:\n")
	fmt.Printf("   BaseURL: %s\n", client.GetEnvironment())
	fmt.Printf("   Environment: %s\n", client.GetEnvironment())
	
	// Verificar que la URL no esté vacía
	if cfg.SUNAT.BaseURL == "" {
		fmt.Println("❌ ERROR: BaseURL está vacía!")
		return
	}
	
	if cfg.SUNAT.BetaURL == "" {
		fmt.Println("❌ ERROR: BetaURL está vacía!")
		return
	}
	
	fmt.Println("✅ URLs configuradas correctamente")
	
	// Probar conexión (opcional)
	fmt.Println("\n🌐 Probando conexión a SUNAT...")
	err = client.Ping()
	if err != nil {
		fmt.Printf("⚠️  Advertencia: No se pudo conectar a SUNAT: %v\n", err)
		fmt.Println("   Esto es normal si no tienes credenciales válidas")
	} else {
		fmt.Println("✅ Conexión exitosa a SUNAT")
	}
	
	fmt.Println("\n🎯 Configuración lista para usar")
} 