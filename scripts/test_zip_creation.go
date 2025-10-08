package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"facturacion_sunat_api_go/internal/services"
)

func main() {
	// Leer el XML firmado de prueba
	xmlPath := "../xml_pruebas/20123456789-01-F001-00000001-signed.xml"
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		log.Fatalf("Error leyendo XML: %v", err)
	}

	fmt.Printf("XML leído: %d bytes\n", len(xmlData))

	// Crear servicio de encoding
	encodingService := services.NewEncodingService()

	// Crear paquete SUNAT
	documentID := "20123456789-01-F001-00000001"
	zipPkg, err := encodingService.ProcessForSUNAT(xmlData, documentID)
	if err != nil {
		log.Fatalf("Error creando paquete SUNAT: %v", err)
	}

	fmt.Printf("Paquete creado:\n")
	fmt.Printf("- Nombre archivo: %s\n", zipPkg.FileName)
	fmt.Printf("- Tamaño ZIP: %d bytes\n", len(zipPkg.ZipContent))
	fmt.Printf("- Tamaño XML: %d bytes\n", len(zipPkg.XMLContent))
	fmt.Printf("- Tamaño Base64: %d caracteres\n", len(zipPkg.Base64Content))

	// Validar el paquete
	if err := zipPkg.ValidatePackage(); err != nil {
		log.Fatalf("Error validando paquete: %v", err)
	}

	fmt.Println("✅ Paquete válido")

	// Validar estructura del ZIP
	if err := encodingService.ValidateZipStructure(zipPkg.ZipContent); err != nil {
		log.Fatalf("Error validando estructura ZIP: %v", err)
	}

	fmt.Println("✅ Estructura ZIP válida")

	// Extraer archivos del ZIP para verificar
	extractedFiles, err := encodingService.ExtractFromZip(zipPkg.ZipContent)
	if err != nil {
		log.Fatalf("Error extrayendo ZIP: %v", err)
	}

	fmt.Printf("Archivos extraídos del ZIP:\n")
	for fileName, content := range extractedFiles {
		fmt.Printf("- %s: %d bytes\n", fileName, len(content))
	}

	// Guardar ZIP de prueba
	zipPath := "../xml_pruebas/test-generated.zip"
	if err := os.WriteFile(zipPath, zipPkg.ZipContent, 0644); err != nil {
		log.Fatalf("Error guardando ZIP: %v", err)
	}

	fmt.Printf("✅ ZIP guardado en: %s\n", zipPath)

	// Verificar que el Base64 se puede decodificar correctamente
	decodedZip, err := encodingService.DecodeFromBase64(zipPkg.Base64Content)
	if err != nil {
		log.Fatalf("Error decodificando Base64: %v", err)
	}

	if len(decodedZip) != len(zipPkg.ZipContent) {
		log.Fatalf("Tamaño de ZIP decodificado no coincide: %d vs %d", len(decodedZip), len(zipPkg.ZipContent))
	}

	fmt.Println("✅ Base64 decodificado correctamente")

	// Verificar que el ZIP decodificado es válido
	if err := encodingService.ValidateZipStructure(decodedZip); err != nil {
		log.Fatalf("Error validando ZIP decodificado: %v", err)
	}

	fmt.Println("✅ ZIP decodificado es válido")

	fmt.Println("\n🎉 Todas las pruebas pasaron exitosamente!")
} 