package main

import (
	"fmt"
	"log"
	"facturacion_sunat_api_go/pkg/certificate"
)

func main() {
	fmt.Println("=== Prueba de Carga de Certificado ===")
	
	// Crear manager de certificados
	certManager := certificate.NewManager()
	
	// Intentar cargar el certificado base64
	cert, privateKey, err := certManager.LoadBase64Certificate("./certs/cert.b64", "./certs/key.b64")
	if err != nil {
		log.Fatalf("Error cargando certificado: %v", err)
	}
	
	fmt.Println("✅ Certificado cargado exitosamente")
	fmt.Printf("📋 Asunto: %s\n", cert.Subject)
	fmt.Printf("🏢 Emisor: %s\n", cert.Issuer)
	fmt.Printf("📅 Válido desde: %s\n", cert.NotBefore.Format("2006-01-02"))
	fmt.Printf("📅 Válido hasta: %s\n", cert.NotAfter.Format("2006-01-02"))
	fmt.Printf("🔢 Número de serie: %s\n", cert.SerialNumber.String())
	
	// Verificar clave privada
	if privateKey != nil {
		fmt.Printf("🔑 Clave RSA de %d bits\n", privateKey.Size()*8)
	} else {
		fmt.Println("❌ No se pudo cargar la clave privada")
	}
	
	// Validar certificado para SUNAT
	if err := certManager.ValidateForSUNAT(cert); err != nil {
		fmt.Printf("⚠️  Advertencia de validación: %v\n", err)
	} else {
		fmt.Println("✅ Certificado válido para SUNAT")
	}
	
	fmt.Println("=== Prueba completada ===")
} 