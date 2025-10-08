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
	fmt.Println("🧪 Probando API después de la corrección...")
	
	// Esperar un momento para que el servidor inicie
	time.Sleep(2 * time.Second)
	
	// Datos de prueba
	testData := map[string]interface{}{
		"tipo": 1,
		"serie": "F001",
		"numero": "00000001",
		"fecha_emision": "2024-01-15T10:30:00Z",
		"tipo_moneda": "PEN",
		"emisor": map[string]interface{}{
			"ruc": "20123456789",
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
				"cantidad": 2,
				"valor_unitario": 2500.00,
				"precio_unitario": 2950.00,
				"tipo_afectacion": 10,
				"valor_venta": 5000.00,
				"valor_total": 5900.00,
				"impuesto_item": []map[string]interface{}{
					{
						"tipo_impuesto": "1000",
						"codigo_impuesto": "IGV",
						"base_imponible": 5000.00,
						"tasa": 18.0,
						"monto_impuesto": 900.00,
					},
				},
			},
		},
		"totales": map[string]interface{}{
			"total_valor_venta": 5000.00,
			"total_venta_gravada": 5000.00,
			"total_impuestos": 900.00,
			"total_precio_venta": 5900.00,
			"importe_total": 5900.00,
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
	client := &http.Client{Timeout: 30 * time.Second}
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
	
	fmt.Printf("📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("📄 Respuesta:\n%s\n", string(body))
	
	// Verificar si el error de URL vacía se solucionó
	if resp.StatusCode == 200 {
		fmt.Println("✅ ¡Éxito! El API funciona correctamente")
		fmt.Println("✅ El error de URL vacía se ha solucionado")
	} else {
		fmt.Println("⚠️  El API respondió pero con un error diferente")
		fmt.Println("   Esto puede ser normal si no tienes certificados válidos")
	}
	
	// Verificar que no aparece el error de URL vacía
	if !bytes.Contains(body, []byte("unsupported protocol scheme")) {
		fmt.Println("✅ El error de URL vacía se ha solucionado correctamente")
	} else {
		fmt.Println("❌ El error de URL vacía persiste")
	}
} 