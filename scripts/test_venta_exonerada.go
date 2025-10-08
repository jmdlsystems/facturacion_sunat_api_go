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
	fmt.Println("🧾 ESCENARIO 2: VENTA EXONERADA")
	fmt.Println("📋 Enviando a SUNAT BETA - Envío REAL")
	fmt.Println("🎯 Objetivo: Recibir CDR certificado de SUNAT BETA")
	
	// Datos de prueba para VENTA EXONERADA
	testData := map[string]interface{}{
		"tipo": 1, // Factura
		"serie": "F001",
		"numero": "00000007",
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
			"razon_social": "CLIENTE EXONERADO S.A.C.",
			"direccion": "AV. CLIENTE 456",
			"distrito": "LIMA",
			"provincia": "LIMA",
			"departamento": "LIMA",
		},
		"items": []map[string]interface{}{
			{
				"id": 1,
				"numero_item": 1,
				"codigo": "P003",
				"descripcion": "Libros Educativos - Venta Exonerada",
				"unidad_medida": "NIU",
				"cantidad": 10,
				"valor_unitario": 25.00,
				"precio_unitario": 25.00,
				"tipo_afectacion": 20, // Exonerado - Operación Onerosa
				"valor_venta": 250.00,
				"valor_total": 250.00,
				"impuesto_item": []map[string]interface{}{
					{
						"tipo_impuesto": "9997",
						"codigo_impuesto": "EXO",
						"base_imponible": 250.00,
						"tasa": 0.0,
						"monto_impuesto": 0.00,
					},
				},
			},
			{
				"id": 2,
				"numero_item": 2,
				"codigo": "P004",
				"descripcion": "Medicamentos - Venta Exonerada",
				"unidad_medida": "NIU",
				"cantidad": 5,
				"valor_unitario": 15.00,
				"precio_unitario": 15.00,
				"tipo_afectacion": 20, // Exonerado - Operación Onerosa
				"valor_venta": 75.00,
				"valor_total": 75.00,
				"impuesto_item": []map[string]interface{}{
					{
						"tipo_impuesto": "9997",
						"codigo_impuesto": "EXO",
						"base_imponible": 75.00,
						"tasa": 0.0,
						"monto_impuesto": 0.00,
					},
				},
			},
		},
		"totales": map[string]interface{}{
			"total_valor_venta": 325.00,
			"total_venta_exonerada": 325.00,
			"total_impuestos": 0.00,
			"total_precio_venta": 325.00,
			"importe_total": 325.00,
		},
	}
	
	// Enviar a SUNAT BETA
	sendToSUNAT(testData, "VENTA EXONERADA")
}

func sendToSUNAT(data map[string]interface{}, scenario string) {
	fmt.Printf("\n🚀 Enviando %s a SUNAT BETA...\n", scenario)
	
	// Convertir a JSON
	jsonData, err := json.Marshal(data)
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
	
	fmt.Printf("📊 Respuesta del API - Status: %d\n", resp.StatusCode)
	
	// Parsear respuesta JSON
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parseando respuesta: %v\n", err)
		return
	}
	
	if resp.StatusCode == 200 {
		fmt.Printf("✅ ¡Éxito! %s enviado a SUNAT BETA\n", scenario)
		
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
		
		fmt.Printf("\n🎯 %s - ENVÍO REAL COMPLETADO:\n", scenario)
		fmt.Println("   - XML firmado enviado a SUNAT BETA")
		fmt.Println("   - Esperando respuesta con CDR certificado")
		fmt.Println("   - Verificar logs del servidor para detalles")
		
	} else {
		fmt.Printf("❌ Error en el API: %s\n", string(body))
	}
} 