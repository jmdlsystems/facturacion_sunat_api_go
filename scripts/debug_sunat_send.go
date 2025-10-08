package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/internal/services"
)

func main() {
	// Simular el proceso completo de envío a SUNAT
	fmt.Println("🔍 Debug: Proceso de envío a SUNAT")
	fmt.Println("==================================")

	// 1. Leer XML firmado
	xmlPath := "../xml_pruebas/20123456789-01-F001-00000001-signed.xml"
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		log.Fatalf("❌ Error leyendo XML: %v", err)
	}
	fmt.Printf("✅ XML leído: %d bytes\n", len(xmlData))

	// 2. Crear servicios
	encodingService := services.NewEncodingService()
	
	// Configuración SUNAT (usar valores por defecto para pruebas)
	sunatConfig := &config.SUNATConfig{
		Username:  "test",
		Password:  "test",
		BaseURL:   "https://test.sunat.gob.pe",
		Timeout:   30,
		MaxRetries: 3,
	}
	sunatService := services.NewSUNATService(sunatConfig, encodingService)

	// 3. Crear paquete SUNAT
	documentID := "20123456789-01-F001-00000001"
	fmt.Printf("📦 Creando paquete para documento: %s\n", documentID)
	
	zipPkg, err := encodingService.ProcessForSUNAT(xmlData, documentID)
	if err != nil {
		log.Fatalf("❌ Error creando paquete SUNAT: %v", err)
	}

	fmt.Printf("✅ Paquete creado:\n")
	fmt.Printf("   - Nombre: %s\n", zipPkg.FileName)
	fmt.Printf("   - ZIP: %d bytes\n", len(zipPkg.ZipContent))
	fmt.Printf("   - XML: %d bytes\n", len(zipPkg.XMLContent))
	fmt.Printf("   - Base64: %d caracteres\n", len(zipPkg.Base64Content))

	// 4. Validar paquete
	fmt.Println("🔍 Validando paquete...")
	if err := zipPkg.ValidatePackage(); err != nil {
		log.Fatalf("❌ Error validando paquete: %v", err)
	}
	fmt.Println("✅ Paquete válido")

	// 5. Mostrar información del paquete
	info := zipPkg.GetPackageInfo()
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	fmt.Printf("📊 Información del paquete:\n%s\n", string(infoJSON))

	// 6. Simular envío (sin enviar realmente)
	fmt.Println("🚀 Simulando envío a SUNAT...")
	fmt.Println("   (No se enviará realmente para evitar errores de conexión)")
	
	// Verificar que el paquete está listo para envío
	if len(zipPkg.ZipContent) == 0 {
		log.Fatalf("❌ ERROR: ZIP está vacío - %d bytes", len(zipPkg.ZipContent))
	}

	if len(zipPkg.XMLContent) == 0 {
		log.Fatalf("❌ ERROR: XML está vacío - %d bytes", len(zipPkg.XMLContent))
	}

	if zipPkg.Base64Content == "" {
		log.Fatalf("❌ ERROR: Base64 está vacío - %d caracteres", len(zipPkg.Base64Content))
	}

	fmt.Println("✅ Paquete listo para envío a SUNAT")
	fmt.Println("🎉 Debug completado exitosamente!")
} 