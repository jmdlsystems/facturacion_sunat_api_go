package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Estructuras para el test completo
type ComprobanteFullTest struct {
	Tipo         string      `json:"tipo"`
	Serie        string      `json:"serie"`
	Numero       string      `json:"numero"`
	TipoMoneda   string      `json:"tipo_moneda"`
	FechaEmision time.Time   `json:"fecha_emision"`
	Emisor       EmisorFull  `json:"emisor"`
	Receptor     ReceptorFull `json:"receptor"`
	Items        []ItemFull  `json:"items"`
}

type EmisorFull struct {
	RUC             string `json:"ruc"`
	RazonSocial     string `json:"razon_social"`
	NombreComercial string `json:"nombre_comercial"`
	TipoDocumento   string `json:"tipo_documento"`
	Direccion       string `json:"direccion"`
	Distrito        string `json:"distrito"`
	Provincia       string `json:"provincia"`
	Departamento    string `json:"departamento"`
	CodigoPostal    string `json:"codigo_postal"`
	CodigoPais      string `json:"codigo_pais"`
	Telefono        string `json:"telefono"`
	Email           string `json:"email"`
}

type ReceptorFull struct {
	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	RazonSocial     string `json:"razon_social"`
	Direccion       string `json:"direccion"`
	Email           string `json:"email"`
}

type ItemFull struct {
	NumeroItem        int     `json:"numero_item"`
	Codigo            string  `json:"codigo"`
	CodigoSUNAT       string  `json:"codigo_sunat"`
	Descripcion       string  `json:"descripcion"`
	UnidadMedida      string  `json:"unidad_medida"`
	Cantidad          float64 `json:"cantidad"`
	ValorUnitario     float64 `json:"valor_unitario"`
	PrecioUnitario    float64 `json:"precio_unitario"`
	DescuentoUnitario float64 `json:"descuento_unitario"`
	TipoAfectacion    int     `json:"tipo_afectacion"`
	ValorVenta        float64 `json:"valor_venta"`
	ValorTotal        float64 `json:"valor_total"`
}

func main() {
	fmt.Println("=== PRUEBA COMPLETA DEL PROCESO SUNAT ===")
	fmt.Println("Generando XML → Firmando → Zipeando → Enviando a SUNAT Beta")
	fmt.Println()

	// Crear comprobante de prueba con datos reales y completos
	comprobante := ComprobanteFullTest{
		Tipo:         "01", // Factura
		Serie:        "F001",
		Numero:       "00000001",
		TipoMoneda:   "PEN",
		FechaEmision: time.Now(),
		Emisor: EmisorFull{
			RUC:             "20103129061",
			RazonSocial:     "EMPRESA DEMO S.A.C.",
			NombreComercial: "EMPRESA DEMO",
			TipoDocumento:   "6", // RUC
			Direccion:       "AV. AREQUIPA 123",
			Distrito:        "LIMA",
			Provincia:       "LIMA",
			Departamento:    "LIMA",
			CodigoPostal:    "15001",
			CodigoPais:      "PE",
			Telefono:        "01-1234567",
			Email:           "demo@empresa.com",
		},
		Receptor: ReceptorFull{
			TipoDocumento:   "6", // RUC
			NumeroDocumento: "20123456789",
			RazonSocial:     "CLIENTE DEMO S.A.C.",
			Direccion:       "AV. TEST 456",
			Email:           "cliente@demo.com",
		},
		Items: []ItemFull{
			{
				NumeroItem:        1,
				Codigo:            "P001",
				CodigoSUNAT:       "25174815",
				Descripcion:       "Producto de prueba para facturación electrónica",
				UnidadMedida:      "NIU",
				Cantidad:          2,
				ValorUnitario:     100.00,
				PrecioUnitario:    118.00, // Con IGV
				DescuentoUnitario: 0,
				TipoAfectacion:    10, // Gravado - Operación Onerosa
				ValorVenta:        200.00,
				ValorTotal:        236.00,
			},
		},
	}

	// Paso 1: Generar XML con firma
	fmt.Println("1️⃣ Generando XML con firma digital...")
	xmlFirmado, err := generarXMLConFirma(comprobante)
	if err != nil {
		fmt.Printf("❌ Error generando XML: %v\n", err)
		return
	}
	fmt.Println("✅ XML generado y firmado exitosamente")

	// Paso 2: Verificar estructura del XML
	fmt.Println("\n2️⃣ Verificando estructura del XML...")
	verificarEstructuraXML(xmlFirmado)

	// Paso 3: Generar ZIP y enviar a SUNAT
	fmt.Println("\n3️⃣ Generando ZIP y enviando a SUNAT Beta...")
	respuesta, err := enviarASUNAT(comprobante)
	if err != nil {
		fmt.Printf("❌ Error enviando a SUNAT: %v\n", err)
		return
	}

	// Paso 4: Procesar respuesta de SUNAT
	fmt.Println("\n4️⃣ Procesando respuesta de SUNAT...")
	procesarRespuestaSUNAT(respuesta)

	fmt.Println("\n=== FIN DEL PROCESO ===")
}

func generarXMLConFirma(comprobante ComprobanteFullTest) (string, error) {
	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	if err != nil {
		return "", fmt.Errorf("error serializando JSON: %v", err)
	}

	// Crear request HTTP
	url := "http://localhost:8080/api/v1/comprobantes/full-process"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creando request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Realizar request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error realizando request: %v", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo respuesta: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Extraer XML firmado
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error parseando respuesta: %v", err)
	}

	xmlFirmado, ok := response["xml_firmado"].(string)
	if !ok {
		return "", fmt.Errorf("no se encontró XML firmado en la respuesta")
	}

	return xmlFirmado, nil
}

func verificarEstructuraXML(xmlContent string) {
	fmt.Println("   Verificando elementos críticos...")

	// Elementos críticos que deben estar presentes
	elementosCriticos := []string{
		"<cbc:ID schemeID=\"6\"",
		"schemeName=\"SUNAT:Identificador de Documento de Identidad\"",
		"schemeAgencyName=\"PE:SUNAT\"",
		"schemeURI=\"urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06\"",
		"<cac:PartyTaxScheme>",
		"<cac:PartyName>",
		"<cac:PartyLegalEntity>",
		"<cac:RegistrationAddress>",
		"<cac:PostalAddress>",
		"<ds:Signature",
		"<ext:UBLExtensions>",
	}

	for _, elemento := range elementosCriticos {
		if strings.Contains(xmlContent, elemento) {
			fmt.Printf("   ✅ %s\n", elemento)
		} else {
			fmt.Printf("   ❌ Falta: %s\n", elemento)
		}
	}

	// Verificar que no haya valores vacíos críticos
	if strings.Contains(xmlContent, "<cbc:ID></cbc:ID>") {
		fmt.Println("   ❌ Encontrado cbc:ID vacío")
	} else {
		fmt.Println("   ✅ No hay cbc:ID vacíos")
	}

	// Verificar encoding UTF-8
	if strings.Contains(xmlContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		fmt.Println("   ✅ Encoding UTF-8 correcto")
	} else {
		fmt.Println("   ❌ Falta encoding UTF-8")
	}
}

func enviarASUNAT(comprobante ComprobanteFullTest) (map[string]interface{}, error) {
	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	if err != nil {
		return nil, fmt.Errorf("error serializando JSON: %v", err)
	}

	// Crear request HTTP para envío completo
	url := "http://localhost:8080/api/v1/comprobantes/send-to-sunat"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Realizar request
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error realizando request: %v", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parsear respuesta
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %v", err)
	}

	return response, nil
}

func procesarRespuestaSUNAT(respuesta map[string]interface{}) {
	fmt.Println("   Procesando respuesta de SUNAT...")

	// Verificar si el envío fue exitoso
	if success, ok := respuesta["success"].(bool); ok && success {
		fmt.Println("   ✅ Envío exitoso a SUNAT")
		
		// Mostrar ticket si está disponible
		if ticket, ok := respuesta["ticket"].(string); ok && ticket != "" {
			fmt.Printf("   📋 Ticket: %s\n", ticket)
		}

		// Mostrar estado si está disponible
		if estado, ok := respuesta["estado_sunat"].(string); ok && estado != "" {
			fmt.Printf("   📊 Estado: %s\n", estado)
		}

		// Mostrar mensaje si está disponible
		if mensaje, ok := respuesta["mensaje"].(string); ok && mensaje != "" {
			fmt.Printf("   💬 Mensaje: %s\n", mensaje)
		}

		// Verificar si hay CDR
		if cdr, ok := respuesta["cdr_sunat"]; ok && cdr != nil {
			fmt.Println("   📄 CDR recibido")
		}

	} else {
		fmt.Println("   ❌ Error en el envío a SUNAT")
		
		// Mostrar detalles del error
		if mensaje, ok := respuesta["mensaje"].(string); ok && mensaje != "" {
			fmt.Printf("   💬 Error: %s\n", mensaje)
		}

		if codigo, ok := respuesta["codigo_error"].(string); ok && codigo != "" {
			fmt.Printf("   🔢 Código: %s\n", codigo)
		}
	}

	// Mostrar timestamp si está disponible
	if timestamp, ok := respuesta["timestamp"].(string); ok && timestamp != "" {
		fmt.Printf("   🕐 Timestamp: %s\n", timestamp)
	}
} 