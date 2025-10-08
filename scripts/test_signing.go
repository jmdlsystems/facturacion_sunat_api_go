package main

import (
	"fmt"
	"log"
	"facturacion_sunat_api_go/internal/services"
	"facturacion_sunat_api_go/pkg/certificate"
)

func main() {
	fmt.Println("=== Prueba de Firma Digital ===")
	
	// Crear servicios necesarios
	certManager := certificate.NewManager()
	ublService := services.NewUBLService()
	signingService := services.NewSigningService(certManager, ublService)
	
	// XML de prueba (ejemplo de factura UBL)
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
         xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">
    <cbc:UBLVersionID>2.1</cbc:UBLVersionID>
    <cbc:CustomizationID>2.0</cbc:CustomizationID>
    <cbc:ID>F001-00000001</cbc:ID>
    <cbc:IssueDate>2024-01-15</cbc:IssueDate>
    <cbc:IssueTime>10:30:00</cbc:IssueTime>
    <cbc:DocumentCurrencyCode>PEN</cbc:DocumentCurrencyCode>
    <cac:AccountingSupplierParty>
        <cac:Party>
            <cac:PartyIdentification>
                <cbc:ID schemeID="6">20103129061</cbc:ID>
            </cac:PartyIdentification>
            <cac:PartyName>
                <cbc:Name>EMPRESA DE PRUEBA SAC</cbc:Name>
            </cac:PartyName>
        </cac:Party>
    </cac:AccountingSupplierParty>
    <cac:AccountingCustomerParty>
        <cac:Party>
            <cac:PartyIdentification>
                <cbc:ID schemeID="1">12345678</cbc:ID>
            </cac:PartyIdentification>
            <cac:PartyName>
                <cbc:Name>CLIENTE DE PRUEBA</cbc:Name>
            </cac:PartyName>
        </cac:Party>
    </cac:AccountingCustomerParty>
    <cac:TaxTotal>
        <cbc:TaxAmount currencyID="PEN">18.00</cbc:TaxAmount>
        <cac:TaxSubtotal>
            <cbc:TaxableAmount currencyID="PEN">100.00</cbc:TaxableAmount>
            <cbc:TaxAmount currencyID="PEN">18.00</cbc:TaxAmount>
            <cac:TaxCategory>
                <cac:TaxScheme>
                    <cbc:ID>1000</cbc:ID>
                    <cbc:Name>IGV</cbc:Name>
                </cac:TaxScheme>
            </cac:TaxCategory>
        </cac:TaxSubtotal>
    </cac:TaxTotal>
    <cac:LegalMonetaryTotal>
        <cbc:LineExtensionAmount currencyID="PEN">100.00</cbc:LineExtensionAmount>
        <cbc:TaxInclusiveAmount currencyID="PEN">118.00</cbc:TaxInclusiveAmount>
        <cbc:PayableAmount currencyID="PEN">118.00</cbc:PayableAmount>
    </cac:LegalMonetaryTotal>
    <cac:InvoiceLine>
        <cbc:ID>1</cbc:ID>
        <cbc:InvoicedQuantity unitCode="NIU">1</cbc:InvoicedQuantity>
        <cbc:LineExtensionAmount currencyID="PEN">100.00</cbc:LineExtensionAmount>
        <cac:Item>
            <cbc:Description>PRODUCTO DE PRUEBA</cbc:Description>
        </cac:Item>
        <cac:Price>
            <cbc:PriceAmount currencyID="PEN">100.00</cbc:PriceAmount>
        </cac:Price>
    </cac:InvoiceLine>
</Invoice>`
	
	fmt.Println("📄 XML de prueba cargado")
	
	// Firmar el documento
	signedXML, err := signingService.SignDocument([]byte(testXML), "./certs/cert.b64", "")
	if err != nil {
		log.Fatalf("❌ Error firmando documento: %v", err)
	}
	
	fmt.Println("✅ Documento firmado exitosamente")
	fmt.Printf("📏 Tamaño del XML firmado: %d bytes\n", len(signedXML))
	
	// Verificar que contiene la firma digital
	xmlStr := string(signedXML)
	if contains(xmlStr, "ds:Signature") {
		fmt.Println("✅ Firma digital encontrada en el XML")
	} else {
		fmt.Println("❌ No se encontró la firma digital")
	}
	
	// Verificar que contiene el certificado
	if contains(xmlStr, "ds:X509Certificate") {
		fmt.Println("✅ Certificado X509 encontrado en la firma")
	} else {
		fmt.Println("❌ No se encontró el certificado X509")
	}
	
	// Verificar la firma
	valid, err := signingService.VerifySignature(signedXML)
	if err != nil {
		fmt.Printf("⚠️  Error verificando firma: %v\n", err)
	} else if valid {
		fmt.Println("✅ Firma digital verificada correctamente")
	} else {
		fmt.Println("❌ La firma digital no es válida")
	}
	
	fmt.Println("=== Prueba completada ===")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())))
} 