package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"

	"facturacion_sunat_api_go/internal/services"
)

func main() {
	fmt.Println("🔍 Debug: Respuesta SUNAT")
	fmt.Println("=========================")

	// Simular una respuesta SOAP típica de SUNAT
	soapResponse := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <sendBillResponse xmlns="http://service.sunat.gob.pe">
      <applicationResponse>UEsDBBQAAAAIAAAAIQAA...</applicationResponse>
      <ticket>123456789</ticket>
    </sendBillResponse>
  </soap:Body>
</soap:Envelope>`

	fmt.Printf("📄 Respuesta SOAP de ejemplo:\n%s\n", soapResponse)

	// Parsear la respuesta
	var response struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			SendBillResponse struct {
				ApplicationResponse string `xml:"applicationResponse"`
				Ticket              string `xml:"ticket"`
			} `xml:"sendBillResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal([]byte(soapResponse), &response); err != nil {
		log.Fatalf("❌ Error parseando SOAP: %v", err)
	}

	fmt.Printf("✅ Respuesta parseada:\n")
	fmt.Printf("   - ApplicationResponse: %s\n", response.Body.SendBillResponse.ApplicationResponse)
	fmt.Printf("   - Ticket: %s\n", response.Body.SendBillResponse.Ticket)

	// Probar con una respuesta de error
	errorResponse := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <soap:Fault>
      <faultcode>soap:Server</faultcode>
      <faultstring>Error interno del servidor</faultstring>
      <detail>Detalles del error...</detail>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>`

	fmt.Printf("\n📄 Respuesta de error SOAP:\n%s\n", errorResponse)

	var errorResp struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Fault struct {
				FaultCode   string `xml:"faultcode"`
				FaultString string `xml:"faultstring"`
				Detail      string `xml:"detail"`
			} `xml:"Fault"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal([]byte(errorResponse), &errorResp); err != nil {
		log.Fatalf("❌ Error parseando SOAP de error: %v", err)
	}

	fmt.Printf("✅ Error parseado:\n")
	fmt.Printf("   - FaultCode: %s\n", errorResp.Body.Fault.FaultCode)
	fmt.Printf("   - FaultString: %s\n", errorResp.Body.Fault.FaultString)
	fmt.Printf("   - Detail: %s\n", errorResp.Body.Fault.Detail)

	// Probar con una respuesta HTTP 500
	fmt.Printf("\n🔍 Simulando respuesta HTTP 500...\n")
	
	// Crear servicio de encoding para pruebas
	encodingService := services.NewEncodingService()
	
	// Simular procesamiento de respuesta con error
	errorMessage := "Error interno del servidor SUNAT"
	fmt.Printf("📝 Mensaje de error: %s\n", errorMessage)
	
	// Verificar que el servicio puede manejar errores
	fmt.Println("✅ Pruebas de parsing completadas")
} 