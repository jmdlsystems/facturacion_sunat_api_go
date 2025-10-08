package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"facturacion_sunat_api_go/internal/models"
	"facturacion_sunat_api_go/internal/services"
	"time"
)

func main() {
	fmt.Println("=== Prueba de Cumplimiento SUNAT UBL 2.1 ===")
	
	// Crear servicios
	ublService := services.NewUBLService()
	conversionService := services.NewConversionService(ublService)
	
	// Crear comprobante de prueba
	comprobante := createTestComprobante()
	
	// Convertir a UBL
	ublStruct, err := conversionService.ConvertToUBL(&comprobante)
	if err != nil {
		log.Fatalf("Error convirtiendo a UBL: %v", err)
	}
	
	// Serializar a XML
	xmlData, err := ublService.SerializeToXML(ublStruct)
	if err != nil {
		log.Fatalf("Error serializando XML: %v", err)
	}
	
	// Formatear para SUNAT
	xmlFormateado, err := ublService.FormatXMLForSUNAT(xmlData)
	if err != nil {
		log.Fatalf("Error formateando XML: %v", err)
	}
	
	// Validar estructura
	if err := ublService.ValidateXMLStructure(xmlFormateado); err != nil {
		log.Fatalf("Error validando estructura XML: %v", err)
	}
	
	fmt.Println("✅ XML generado exitosamente")
	fmt.Println("✅ Estructura UBL 2.1 válida")
	fmt.Println("✅ Namespaces correctos")
	fmt.Println("✅ Elementos obligatorios presentes")
	
	// Mostrar XML generado
	fmt.Println("\n=== XML Generado ===")
	fmt.Println(string(xmlFormateado))
	
	// Verificar elementos específicos SUNAT
	verifySUNATElements(string(xmlFormateado))
}

func createTestComprobante() models.Comprobante {
	fechaEmision := time.Now()
	
	return models.Comprobante{
		ID:           "test-001",
		Tipo:         models.TipoFactura,
		Serie:        "F001",
		Numero:       "00000001",
		FechaEmision: fechaEmision,
		TipoMoneda:   models.SUNATConstants.PENCurrency,
		Emisor: models.Emisor{
			RUC:           "20123456789",
			RazonSocial:   "EMPRESA DEMO S.A.C.",
			TipoDocumento: models.SUNATConstants.RUCDocumentType,
			Direccion:     "AV. DEMO 123",
			Distrito:      "LIMA",
			Provincia:     "LIMA",
			Departamento:  "LIMA",
			CodigoPais:    "PE",
		},
		Receptor: models.Receptor{
			TipoDocumento:    models.SUNATConstants.RUCDocumentType,
			NumeroDocumento:  "20123456780",
			RazonSocial:      "CLIENTE DEMO S.A.C.",
			Direccion:        "AV. CLIENTE 456",
			Distrito:         "LIMA",
			Provincia:        "LIMA",
			Departamento:     "LIMA",
		},
		Items: []models.Item{
			{
				ID:              1,
				NumeroItem:      1,
				Codigo:          "P001",
				Descripcion:     "Laptop HP Pavilion",
				UnidadMedida:    models.SUNATConstants.NIUUnit,
				Cantidad:        2,
				ValorUnitario:   2500.00,
				PrecioUnitario:  2950.00,
				TipoAfectacion:  models.GravadoOneroso,
				ValorVenta:      5000.00,
				ValorTotal:      5900.00,
				ImpuestoItem: []models.ImpuestoItem{
					{
						TipoImpuesto:   models.SUNATConstants.IGVCode,
						CodigoImpuesto: "IGV",
						BaseImponible:  5000.00,
						Tasa:           18.0,
						MontoImpuesto:  900.00,
					},
				},
			},
		},
		Totales: models.Totales{
			TotalValorVenta:     5000.00,
			TotalVentaGravada:   5000.00,
			TotalImpuestos:      900.00,
			TotalPrecioVenta:    5900.00,
			ImporteTotal:        5900.00,
		},
		Impuestos: []models.Impuesto{
			{
				TipoImpuesto:   models.SUNATConstants.IGVCode,
				CodigoImpuesto: "IGV",
				BaseImponible:  5000.00,
				Tasa:           18.0,
				MontoImpuesto:  900.00,
			},
		},
	}
}

func verifySUNATElements(xmlStr string) {
	fmt.Println("\n=== Verificación de Elementos SUNAT ===")
	
	// Verificar namespaces requeridos
	requiredNamespaces := []string{
		"urn:oasis:names:specification:ubl:schema:xsd:Invoice-2",
		"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2",
		"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2",
		"urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2",
	}
	
	for _, ns := range requiredNamespaces {
		if contains(xmlStr, ns) {
			fmt.Printf("✅ Namespace presente: %s\n", ns)
		} else {
			fmt.Printf("❌ Namespace faltante: %s\n", ns)
		}
	}
	
	// Verificar elementos obligatorios
	requiredElements := []string{
		"<cbc:UBLVersionID>2.1</cbc:UBLVersionID>",
		"<cbc:CustomizationID>2.0</cbc:CustomizationID>",
		"<cbc:ID>F001-00000001</cbc:ID>",
		"<cbc:IssueDate>",
		"<cbc:InvoiceTypeCode>01</cbc:InvoiceTypeCode>",
		"<cbc:DocumentCurrencyCode>PEN</cbc:DocumentCurrencyCode>",
		"<cbc:LineCountNumeric>1</cbc:LineCountNumeric>",
		"<ext:UBLExtensions>",
		"<cac:AccountingSupplierParty>",
		"<cac:AccountingCustomerParty>",
		"<cac:TaxTotal>",
		"<cac:LegalMonetaryTotal>",
		"<cac:InvoiceLine>",
	}
	
	for _, element := range requiredElements {
		if contains(xmlStr, element) {
			fmt.Printf("✅ Elemento presente: %s\n", element)
		} else {
			fmt.Printf("❌ Elemento faltante: %s\n", element)
		}
	}
	
	// Verificar atributos SUNAT
	sunatAttributes := []string{
		`schemeID="6"`,
		`schemeName="SUNAT:Identificador de Documento de Identidad"`,
		`schemeAgencyName="PE:SUNAT"`,
		`currencyID="PEN"`,
		`unitCode="NIU"`,
	}
	
	for _, attr := range sunatAttributes {
		if contains(xmlStr, attr) {
			fmt.Printf("✅ Atributo SUNAT presente: %s\n", attr)
		} else {
			fmt.Printf("❌ Atributo SUNAT faltante: %s\n", attr)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
} 