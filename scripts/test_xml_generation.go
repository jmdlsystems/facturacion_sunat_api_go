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
	Tipo         string    `json:"tipo"`
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
	Direccion       string `json:"direccion"`
	Distrito        string `json:"distrito"`
	Provincia       string `json:"provincia"`
	Departamento    string `json:"departamento"`
	Pais            string `json:"pais"`
}

type ReceptorTest struct {
	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	RazonSocial     string `json:"razon_social"`
	Direccion       string `json:"direccion"`
	Distrito        string `json:"distrito"`
	Provincia       string `json:"provincia"`
	Departamento    string `json:"departamento"`
	Pais            string `json:"pais"`
}

type ItemTest struct {
	Codigo         string        `json:"codigo"`
	Descripcion    string        `json:"descripcion"`
	Cantidad       float64       `json:"cantidad"`
	PrecioUnitario float64       `json:"precio_unitario"`
	Impuestos      []ImpuestoItemTest `json:"impuestos"`
}

type ImpuestoItemTest struct {
	Tipo       string  `json:"tipo"`
	Porcentaje float64 `json:"porcentaje"`
}

func main() {
	fmt.Println("=== Prueba de Generación de XML sin Firma ===")

	// Crear comprobante de prueba
	comprobante := ComprobanteTest{
		Tipo:         "01", // Factura
		Serie:        "F001",
		Numero:       "00000001",
		TipoMoneda:   "PEN",
		FechaEmision: time.Now(),
		Emisor: EmisorTest{
			RUC:             "20123456789",
			RazonSocial:     "EMPRESA DE PRUEBA SAC",
			NombreComercial: "EMPRESA DE PRUEBA",
			Direccion:       "AV. AREQUIPA 123",
			Distrito:        "LIMA",
			Provincia:       "LIMA",
			Departamento:    "LIMA",
			Pais:            "PE",
		},
		Receptor: ReceptorTest{
			TipoDocumento:   "1", // DNI
			NumeroDocumento: "12345678",
			RazonSocial:     "CLIENTE DE PRUEBA",
			Direccion:       "AV. TEST 456",
			Distrito:        "LIMA",
			Provincia:       "LIMA",
			Departamento:    "LIMA",
			Pais:            "PE",
		},
		Items: []ItemTest{
			{
				Codigo:         "PROD001",
				Descripcion:    "PRODUCTO DE PRUEBA",
				Cantidad:       1,
				PrecioUnitario: 100.00,
				Impuestos: []ImpuestoItemTest{
					{
						Tipo:       "1000", // IGV
						Porcentaje: 18.0,
					},
				},
			},
		},
	}

	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	if err != nil {
		fmt.Printf("Error serializando JSON: %v\n", err)
		return
	}

	// Crear request HTTP
	url := "http://localhost:8080/api/v1/utils/generate-xml"
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
		fmt.Printf("Document ID: %s\n", response["document_id"])
		fmt.Printf("File Name: %s\n", response["file_name"])

		// Mostrar parte del XML generado
		if xmlContent, ok := response["xml_content"].(string); ok {
			fmt.Println("\n=== XML Generado (primeras 500 caracteres) ===")
			if len(xmlContent) > 500 {
				fmt.Println(xmlContent[:500] + "...")
			} else {
				fmt.Println(xmlContent)
			}
		}

		// Mostrar totales calculados
		if totals, ok := response["totals"].(map[string]interface{}); ok {
			fmt.Println("\n=== Totales Calculados ===")
			for key, value := range totals {
				fmt.Printf("%s: %v\n", key, value)
			}
		}
	} else {
		fmt.Println("Error en la respuesta del servidor")
	}
}

// Función para probar diferentes tipos de comprobantes
func testDifferentTypes() {
	fmt.Println("\n=== Probando Diferentes Tipos de Comprobantes ===")

	types := []struct {
		name string
		tipo string
	}{
		{"Factura", "01"},
		{"Boleta", "03"},
		{"Nota de Crédito", "07"},
		{"Nota de Débito", "08"},
	}

	for _, t := range types {
		fmt.Printf("\n--- Probando %s ---\n", t.name)
		
		comprobante := ComprobanteTest{
			Tipo:         t.tipo,
			Serie:        "F001",
			Numero:       "00000001",
			TipoMoneda:   "PEN",
			FechaEmision: time.Now(),
			Emisor: EmisorTest{
				RUC:         "20123456789",
				RazonSocial: "EMPRESA DE PRUEBA SAC",
			},
			Receptor: ReceptorTest{
				TipoDocumento:   "1",
				NumeroDocumento: "12345678",
				RazonSocial:     "CLIENTE DE PRUEBA",
			},
			Items: []ItemTest{
				{
					Codigo:         "PROD001",
					Descripcion:    "PRODUCTO DE PRUEBA",
					Cantidad:       1,
					PrecioUnitario: 100.00,
					Impuestos: []ImpuestoItemTest{
						{
							Tipo:       "1000",
							Porcentaje: 18.0,
						},
					},
				},
			},
		}

		// Convertir a JSON
		jsonData, err := json.Marshal(comprobante)
		if err != nil {
			fmt.Printf("Error serializando JSON: %v\n", err)
			continue
		}

		// Crear request HTTP
		url := "http://localhost:8080/api/v1/utils/generate-xml"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error creando request: %v\n", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Realizar request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error realizando request: %v\n", err)
			continue
		}

		fmt.Printf("Status: %d\n", resp.StatusCode)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("✓ %s generado exitosamente\n", t.name)
		} else {
			fmt.Printf("✗ Error generando %s\n", t.name)
		}
	}
} 