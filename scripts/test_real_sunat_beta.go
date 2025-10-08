package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🚀 Probando envío REAL a SUNAT BETA...")
	fmt.Println("📋 Configuración:")
	fmt.Println("   - URL: https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService")
	fmt.Println("   - Usuario: 20103129061MODDATOS")
	fmt.Println("   - Modo: ENVÍO REAL (force_real_send: true)")
	fmt.Println("   - Objetivo: Recibir CDR certificado de SUNAT BETA")
	
	// Datos de prueba para factura
	testData := map[string]interface{}{
		"tipo": 1, // Factura
		"serie": "F001",
		"numero": "00000003",
		"fecha_emision": "2024-01-15T10:30:00Z",
		"tipo_moneda": "PEN",
		"emisor": map[string]interface{}{
			"ruc": "20103129061",
			"razon_social": "EMPRESA DEMO S.A.C.",
			"tipo_documento": "6",
			"direccion": "AV. DEMO 123",
			"distrito": "LIMA",
			"provincia": "LIMA",
			"departamento": "LIMA",
			"codigo_pais": "PE",
		},
		"receptor": map[string]interface{}{
			"tipo_documento": "6",
			"numero_documento": "20123456780",
			"razon_social": "CLIENTE DEMO S.A.C.",
			"direccion": "AV. CLIENTE 456",
			"distrito": "LIMA",
			"provincia": "LIMA",
			"departamento": "LIMA",
		},
		"items": []map[string]interface{}{
			{
				"id": 1,
				"numero_item": 1,
				"codigo": "P001",
				"descripcion": "Laptop HP Pavilion",
				"unidad_medida": "NIU",
				"cantidad": 1,
				"valor_unitario": 2500.00,
				"precio_unitario": 2950.00,
				"tipo_afectacion": 10,
				"valor_venta": 2500.00,
				"valor_total": 2950.00,
				"impuesto_item": []map[string]interface{}{
					{
						"tipo_impuesto": "1000",
						"codigo_impuesto": "IGV",
						"base_imponible": 2500.00,
						"tasa": 18.0,
						"monto_impuesto": 450.00,
					},
				},
			},
		},
		"totales": map[string]interface{}{
			"total_valor_venta": 2500.00,
			"total_venta_gravada": 2500.00,
			"total_impuestos": 450.00,
			"total_precio_venta": 2950.00,
			"importe_total": 2950.00,
		},
	}
	
	// Convertir a JSON
	jsonData, err := json.Marshal(testData)
	if err != nil {
		fmt.Printf("❌ Error serializando JSON: %v\n", err)
		return
	}
	
	// Crear request
	url := "http://localhost:8081/api/v1/comprobantes/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error creando request: %v\n", err)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Realizar request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error realizando request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error leyendo respuesta: %v\n", err)
		return
	}
	
	fmt.Printf("\n📊 Respuesta del API - Status: %d\n", resp.StatusCode)
	
	// Parsear respuesta JSON
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parseando respuesta: %v\n", err)
		return
	}
	
	if resp.StatusCode == 200 {
		fmt.Println("✅ ¡Éxito! Comprobante creado y enviado a SUNAT BETA")
		
		// Extraer información importante
		if id, ok := response["id"].(string); ok {
			fmt.Printf("📋 ID del comprobante: %s\n", id)
		}
		
		if message, ok := response["message"].(string); ok {
			fmt.Printf("💬 Mensaje: %s\n", message)
		}
		
		// Verificar si hay XML firmado
		if xmlFirmado, ok := response["xml_firmado"].(string); ok && len(xmlFirmado) > 0 {
			fmt.Println("✅ XML firmado generado correctamente")
		}
		
		// Verificar si hay ZIP
		if zipBase64, ok := response["zip_base64"].(string); ok && len(zipBase64) > 0 {
			fmt.Println("✅ ZIP generado correctamente")
		}
		
		// Verificar si hay ticket de SUNAT
		if ticket, ok := response["ticket"].(string); ok && len(ticket) > 0 {
			fmt.Printf("🎫 Ticket SUNAT: %s\n", ticket)
		}
		
		// Verificar si hay CDR
		if cdr, ok := response["cdr"].(string); ok && len(cdr) > 0 {
			fmt.Println("📄 CDR recibido de SUNAT BETA")
		}
		
		fmt.Println("\n🚀 Envío REAL a SUNAT BETA:")
		fmt.Println("   - XML firmado enviado a SUNAT BETA")
		fmt.Println("   - Esperando respuesta con CDR certificado")
		fmt.Println("   - Verificar logs del servidor para detalles")
		
	} else {
		fmt.Printf("❌ Error en el API: %s\n", string(body))
	}
	
	fmt.Println("\n📝 Notas importantes:")
	fmt.Println("   1. El XML se envía REALMENTE a SUNAT BETA")
	fmt.Println("   2. SUNAT BETA procesará el documento")
	fmt.Println("   3. Se recibirá un CDR certificado de respuesta")
	fmt.Println("   4. Verificar logs del servidor para ver la respuesta SOAP")
	fmt.Println("   5. El CDR contendrá el estado de aceptación/rechazo")
} 