package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ComprobanteTest representa la estructura para pruebas
type ComprobanteTest struct {
	Tipo         int       `json:"tipo"`
	Serie        string    `json:"serie"`
	Numero       string    `json:"numero"`
	TipoMoneda   string    `json:"tipo_moneda"`
	FechaEmision time.Time `json:"fecha_emision"`
	Emisor       EmisorTest `json:"emisor"`
	Receptor     ReceptorTest `json:"receptor"`
	Items        []ItemTest `json:"items"`
}

type EmisorTest struct {
	RUC             string `json:"ruc"`
	RazonSocial     string `json:"razon_social"`
	NombreComercial string `json:"nombre_comercial"`
	TipoDocumento   string `json:"tipo_documento"`
	Direccion       string `json:"direccion"`
	Distrito        string `json:"distrito"`
	Provincia       string `json:"provincia"`
	Departamento    string `json:"departamento"`
	CodigoPais      string `json:"codigo_pais"`
	Telefono        string `json:"telefono,omitempty"`
	Email           string `json:"email,omitempty"`
}

type ReceptorTest struct {
	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	RazonSocial     string `json:"razon_social"`
	Direccion       string `json:"direccion,omitempty"`
	Email           string `json:"email,omitempty"`
}

type ItemTest struct {
	ID             int                `json:"id"`
	NumeroItem     int                `json:"numero_item"`
	Codigo         string             `json:"codigo"`
	Descripcion    string             `json:"descripcion"`
	UnidadMedida   string             `json:"unidad_medida"`
	Cantidad       float64            `json:"cantidad"`
	ValorUnitario  float64            `json:"valor_unitario"`
	PrecioUnitario float64            `json:"precio_unitario"`
	TipoAfectacion int                `json:"tipo_afectacion"`
	ImpuestoItem   []ImpuestoItemTest `json:"impuesto_item"`
	ValorVenta     float64            `json:"valor_venta"`
	ValorTotal     float64            `json:"valor_total"`
}

type ImpuestoItemTest struct {
	TipoImpuesto   string  `json:"tipo_impuesto"`
	CodigoImpuesto string  `json:"codigo_impuesto"`
	BaseImponible  float64 `json:"base_imponible"`
	Tasa           float64 `json:"tasa"`
	MontoImpuesto  float64 `json:"monto_impuesto"`
}

func main() {
	fmt.Println("=== Prueba de Creación de Comprobante ===")

	// Crear comprobante de prueba válido
	comprobante := ComprobanteTest{
		Tipo:         1, // Factura
		Serie:        "F001",
		Numero:       "00000001",
		TipoMoneda:   "PEN",
		FechaEmision: time.Now(),
		Emisor: EmisorTest{
			RUC:             "20123456789",
			RazonSocial:     "EMPRESA DEMO S.A.C.",
			NombreComercial: "EMPRESA DEMO",
			TipoDocumento:   "6",
			Direccion:       "AV. AREQUIPA 123",
			Distrito:        "LIMA",
			Provincia:       "LIMA",
			Departamento:    "LIMA",
			CodigoPais:      "PE",
			Telefono:        "01-1234567",
			Email:           "contacto@empresademo.com",
		},
		Receptor: ReceptorTest{
			TipoDocumento:   "6",
			NumeroDocumento: "12345678",
			RazonSocial:     "CLIENTE DEMO S.A.C.",
			Direccion:       "AV. TEST 456",
			Email:           "cliente@clientedemo.com",
		},
		Items: []ItemTest{
			{
				ID:             1,
				NumeroItem:     1,
				Codigo:         "P001",
				Descripcion:    "Producto de prueba",
				UnidadMedida:   "NIU",
				Cantidad:       2,
				ValorUnitario:  100,
				PrecioUnitario: 118,
				TipoAfectacion: 10,
				ImpuestoItem: []ImpuestoItemTest{
					{
						TipoImpuesto:   "1000",
						CodigoImpuesto: "IGV",
						BaseImponible:  200,
						Tasa:           18,
						MontoImpuesto:  36,
					},
				},
				ValorVenta: 200,
				ValorTotal: 236,
			},
		},
	}

	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	if err != nil {
		fmt.Printf("Error serializando JSON: %v\n", err)
		return
	}

	fmt.Printf("JSON enviado:\n%s\n\n", string(jsonData))

	// Crear request HTTP
	url := "http://localhost:8080/api/v1/comprobantes/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creando request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Realizar request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error realizando request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error leyendo respuesta: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body:\n%s\n", string(body))

	// Parsear respuesta si es exitosa
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			fmt.Printf("Error parseando respuesta: %v\n", err)
			return
		}

		fmt.Println("\n=== Detalles de la Respuesta ===")
		fmt.Printf("Mensaje: %s\n", response["message"])
		fmt.Printf("ID: %s\n", response["id"])
		fmt.Printf("File Name: %s\n", response["file_name"])

		// Mostrar parte del XML generado
		if xmlContent, ok := response["xml_ubl"].(string); ok {
			fmt.Println("\n=== XML Generado (primeras 500 caracteres) ===")
			if len(xmlContent) > 500 {
				fmt.Println(xmlContent[:500] + "...")
			} else {
				fmt.Println(xmlContent)
			}

			// Verificar que contiene el RUC
			if contains(xmlContent, "20123456789") {
				fmt.Println("✅ RUC encontrado en el XML")
			} else {
				fmt.Println("❌ RUC NO encontrado en el XML")
			}

			// Verificar que NO contiene firma
			if contains(xmlContent, "<ds:Signature") {
				fmt.Println("❌ XML contiene firma (no debería)")
			} else {
				fmt.Println("✅ XML NO contiene firma (correcto)")
			}
		}
	} else {
		fmt.Println("Error en la respuesta del servidor")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
} 