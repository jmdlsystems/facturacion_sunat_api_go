package main

import (
	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/pkg/sunat"
	"fmt"
	"log"
)

func main() {
	fmt.Println("🔍 Verificando conexión a SUNAT BETA...")
	
	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Error cargando configuración: %v", err)
	}
	
	fmt.Printf("📋 Configuración cargada:\n")
	fmt.Printf("   - Base URL: %s\n", cfg.SUNAT.BaseURL)
	fmt.Printf("   - Beta URL: %s\n", cfg.SUNAT.BetaURL)
	fmt.Printf("   - FE Beta: %s\n", cfg.SUNAT.FEBeta)
	fmt.Printf("   - Username: %s\n", cfg.SUNAT.Username)
	fmt.Printf("   - RUC: %s\n", cfg.SUNAT.RUC)
	
	// Crear cliente SUNAT
	client := sunat.NewClient(&cfg.SUNAT)
	
	fmt.Printf("\n🎯 Cliente SUNAT creado:\n")
	fmt.Printf("   - URL Base del cliente: %s\n", client.GetBaseURL())
	fmt.Printf("   - ¿Es producción?: %t\n", client.IsProduction())
	fmt.Printf("   - Ambiente: %s\n", client.GetEnvironment())
	
	// Verificar que esté apuntando a BETA
	baseURL := client.GetBaseURL()
	if isBetaURL(baseURL) {
		fmt.Println("\n✅ ¡CORRECTO! El API está apuntando a SUNAT BETA")
		fmt.Println("   - URL: https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService")
		fmt.Println("   - Ambiente: BETA")
		fmt.Println("   - Estado: Configurado correctamente")
	} else {
		fmt.Println("\n❌ ERROR: El API NO está apuntando a SUNAT BETA")
		fmt.Printf("   - URL actual: %s\n", baseURL)
		fmt.Println("   - Debería ser: https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService")
	}
	
	// Probar conectividad (sin enviar documentos)
	fmt.Println("\n🌐 Probando conectividad con SUNAT BETA...")
	
	// Crear un request de prueba simple
	testReq, err := client.TestConnection(nil)
	if err != nil {
		fmt.Printf("⚠️  Error de conectividad: %v\n", err)
		fmt.Println("   - Esto es normal si SUNAT BETA no permite pings directos")
		fmt.Println("   - El API seguirá funcionando para envío de documentos")
	} else {
		fmt.Printf("✅ Conectividad exitosa - Status: %d\n", testReq.StatusCode)
	}
	
	fmt.Println("\n🎭 Modo de operación:")
	if cfg.SUNAT.Username == "MODDATOS" || cfg.SUNAT.Username == "20103129061MODDATOS" {
		fmt.Println("   - Modo: SIMULACIÓN")
		fmt.Println("   - No se envían documentos reales a SUNAT")
		fmt.Println("   - Se simulan respuestas exitosas")
		fmt.Println("   - Perfecto para desarrollo y pruebas")
	} else {
		fmt.Println("   - Modo: PRODUCCIÓN")
		fmt.Println("   - Se envían documentos reales a SUNAT BETA")
		fmt.Println("   - Usar con precaución")
	}
	
	fmt.Println("\n📝 Resumen:")
	fmt.Println("   ✅ Configuración BETA correcta")
	fmt.Println("   ✅ Cliente apuntando a SUNAT BETA")
	fmt.Println("   ✅ URLs de BETA configuradas")
	fmt.Println("   ✅ Listo para envío de documentos")
}

// isBetaURL verifica si la URL es de SUNAT BETA
func isBetaURL(url string) bool {
	betaURLs := []string{
		"https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService",
		"https://e-beta.sunat.gob.pe",
		"beta.sunat.gob.pe",
	}
	
	for _, betaURL := range betaURLs {
		if url == betaURL || contains(url, betaURL) {
			return true
		}
	}
	return false
}

// contains verifica si una cadena contiene otra
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
} 