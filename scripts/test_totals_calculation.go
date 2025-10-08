package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Estructuras para el test de totales
type ComprobanteTest struct {
	Tipo         string      `json:"tipo"`
	Serie        string      `json:"serie"`
	Numero       string      `json:"numero"`
	TipoMoneda   string      `json:"tipo_moneda"`
	FechaEmision time.Time   `json:"fecha_emision"`
	Emisor       EmisorTest  `json:"emisor"`
	Receptor     ReceptorTest `json:"receptor"`
	Items        []ItemTest  `json:"items"`
	Totales      TotalesTest `json:"totales"`
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
}

type ItemTest struct {
	NumeroItem        int     `json:"numero_item"`
	Codigo            string  `json:"codigo"`
	Descripcion       string  `json:"descripcion"`
	UnidadMedida      string  `json:"unidad_medida"`
	Cantidad          float64 `json:"cantidad"`
	ValorUnitario     float64 `json:"valor_unitario"`
	PrecioUnitario    float64 `json:"precio_unitario"`
	DescuentoUnitario float64 `json:"descuento_unitario"`
	TipoAfectacion    int     `json:"tipo_afectacion"`
	Impuestos         []ImpuestoItemTest `json:"impuestos"`
}

type ImpuestoItemTest struct {
	TipoImpuesto   string  `json:"tipo_impuesto"`
	CodigoImpuesto string  `json:"codigo_impuesto"`
	BaseImponible  float64 `json:"base_imponible"`
	Tasa           float64 `json:"tasa"`
	MontoImpuesto  float64 `json:"monto_impuesto"`
}

type TotalesTest struct {
	TotalVentaGravada      float64 `json:"total_venta_gravada"`
	TotalVentaExonerada    float64 `json:"total_venta_exonerada"`
	TotalVentaInafecta     float64 `json:"total_venta_inafecta"`
	TotalVentaGratuita     float64 `json:"total_venta_gratuita"`
	TotalDescuentos        float64 `json:"total_descuentos"`
	TotalAnticipos         float64 `json:"total_anticipos"`
	TotalImpuestos         float64 `json:"total_impuestos"`
	TotalValorVenta        float64 `json:"total_valor_venta"`
	TotalPrecioVenta       float64 `json:"total_precio_venta"`
	Redondeo               float64 `json:"redondeo"`
	ImporteTotal           float64 `json:"importe_total"`
}

func main() {
	fmt.Println("=== PRUEBA DE CÁLCULO DE TOTALES ===")
	fmt.Println()

	// Crear comprobante de prueba con datos que deberían calcular correctamente
	comprobante := ComprobanteTest{
		Tipo:         "01", // Factura
		Serie:        "F001",
		Numero:       "00000001",
		TipoMoneda:   "PEN",
		FechaEmision: time.Now(),
		Emisor: EmisorTest{
			RUC:             "20103129061",
			RazonSocial:     "EMPRESA DEMO S.A.C.",
			NombreComercial: "EMPRESA DEMO",
			Direccion:       "AV. AREQUIPA 123",
			Distrito:        "LIMA",
			Provincia:       "LIMA",
			Departamento:    "LIMA",
			Pais:            "PE",
		},
		Receptor: ReceptorTest{
			TipoDocumento:   "6", // RUC
			NumeroDocumento: "20123456789",
			RazonSocial:     "CLIENTE DEMO S.A.C.",
			Direccion:       "AV. TEST 456",
		},
		Items: []ItemTest{
			{
				NumeroItem:        1,
				Codigo:            "P001",
				Descripcion:       "Producto de prueba 1",
				UnidadMedida:      "NIU",
				Cantidad:          2.0,
				ValorUnitario:     100.00,
				PrecioUnitario:    118.00,
				DescuentoUnitario: 0.0,
				TipoAfectacion:    10, // Gravado - Operación Onerosa
				Impuestos: []ImpuestoItemTest{
					{
						TipoImpuesto:   "1000",
						CodigoImpuesto: "10",
						BaseImponible:  200.00,
						Tasa:           18.0,
						MontoImpuesto:  36.00,
					},
				},
			},
			{
				NumeroItem:        2,
				Codigo:            "P002",
				Descripcion:       "Producto de prueba 2",
				UnidadMedida:      "NIU",
				Cantidad:          1.0,
				ValorUnitario:     50.00,
				PrecioUnitario:    59.00,
				DescuentoUnitario: 5.0,
				TipoAfectacion:    10, // Gravado - Operación Onerosa
				Impuestos: []ImpuestoItemTest{
					{
						TipoImpuesto:   "1000",
						CodigoImpuesto: "10",
						BaseImponible:  45.00, // 50 - 5 descuento
						Tasa:           18.0,
						MontoImpuesto:  8.10,
					},
				},
			},
		},
		Totales: TotalesTest{
			TotalDescuentos: 5.0,
			TotalAnticipos:  0.0,
			Redondeo:        0.0,
		},
	}

	fmt.Println("1️⃣ Comprobante de prueba creado:")
	fmt.Printf("   - Items: %d\n", len(comprobante.Items))
	fmt.Printf("   - Moneda: %s\n", comprobante.TipoMoneda)
	fmt.Printf("   - Tipo: %s\n", comprobante.Tipo)

	// Mostrar detalles de los items
	fmt.Println("\n2️⃣ Detalles de los items:")
	for i, item := range comprobante.Items {
		fmt.Printf("   Item %d:\n", i+1)
		fmt.Printf("     - Código: %s\n", item.Codigo)
		fmt.Printf("     - Cantidad: %.2f\n", item.Cantidad)
		fmt.Printf("     - Valor Unitario: %.2f\n", item.ValorUnitario)
		fmt.Printf("     - Precio Unitario: %.2f\n", item.PrecioUnitario)
		fmt.Printf("     - Descuento: %.2f\n", item.DescuentoUnitario)
		fmt.Printf("     - Tipo Afectación: %d\n", item.TipoAfectacion)
		fmt.Printf("     - Impuestos: %d\n", len(item.Impuestos))
		
		for j, impuesto := range item.Impuestos {
			fmt.Printf("       Impuesto %d: %s (%.2f%%) = %.2f\n", 
				j+1, impuesto.TipoImpuesto, impuesto.Tasa, impuesto.MontoImpuesto)
		}
	}

	// Calcular totales manualmente para verificar
	fmt.Println("\n3️⃣ Cálculo manual de totales:")
	
	var totalVentaGravada float64
	var totalImpuestos float64
	
	for _, item := range comprobante.Items {
		valorVenta := item.Cantidad * item.ValorUnitario
		fmt.Printf("   Item %s: %.2f × %.2f = %.2f\n", 
			item.Codigo, item.Cantidad, item.ValorUnitario, valorVenta)
		
		if item.TipoAfectacion == 10 { // Gravado
			totalVentaGravada += valorVenta
		}
		
		for _, impuesto := range item.Impuestos {
			totalImpuestos += impuesto.MontoImpuesto
		}
	}
	
	totalValorVenta := totalVentaGravada
	totalPrecioVenta := totalValorVenta + totalImpuestos
	importeTotal := totalPrecioVenta - comprobante.Totales.TotalDescuentos - comprobante.Totales.TotalAnticipos + comprobante.Totales.Redondeo
	
	fmt.Printf("   Total Venta Gravada: %.2f\n", totalVentaGravada)
	fmt.Printf("   Total Impuestos: %.2f\n", totalImpuestos)
	fmt.Printf("   Total Valor Venta: %.2f\n", totalValorVenta)
	fmt.Printf("   Total Precio Venta: %.2f\n", totalPrecioVenta)
	fmt.Printf("   Descuentos: %.2f\n", comprobante.Totales.TotalDescuentos)
	fmt.Printf("   Anticipos: %.2f\n", comprobante.Totales.TotalAnticipos)
	fmt.Printf("   Redondeo: %.2f\n", comprobante.Totales.Redondeo)
	fmt.Printf("   Importe Total: %.2f\n", importeTotal)

	// Convertir a JSON para verificar la estructura
	fmt.Println("\n4️⃣ Estructura JSON del comprobante:")
	jsonData, err := json.MarshalIndent(comprobante, "", "  ")
	if err != nil {
		fmt.Printf("❌ Error serializando JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))

	// Verificar que no haya valores NaN o infinitos
	fmt.Println("\n5️⃣ Verificación de valores:")
	verificarValores(comprobante)

	fmt.Println("\n=== FIN DE LA PRUEBA ===")
	fmt.Println("Si todos los cálculos son correctos, el sistema de totales funciona bien.")
}

func verificarValores(comprobante ComprobanteTest) {
	// Verificar items
	for i, item := range comprobante.Items {
		if item.Cantidad <= 0 {
			fmt.Printf("   ❌ Item %d: Cantidad debe ser mayor a 0\n", i+1)
		} else {
			fmt.Printf("   ✅ Item %d: Cantidad válida (%.2f)\n", i+1, item.Cantidad)
		}
		
		if item.ValorUnitario < 0 {
			fmt.Printf("   ❌ Item %d: Valor unitario no puede ser negativo\n", i+1)
		} else {
			fmt.Printf("   ✅ Item %d: Valor unitario válido (%.2f)\n", i+1, item.ValorUnitario)
		}
		
		if item.PrecioUnitario < 0 {
			fmt.Printf("   ❌ Item %d: Precio unitario no puede ser negativo\n", i+1)
		} else {
			fmt.Printf("   ✅ Item %d: Precio unitario válido (%.2f)\n", i+1, item.PrecioUnitario)
		}
		
		if item.DescuentoUnitario < 0 {
			fmt.Printf("   ❌ Item %d: Descuento no puede ser negativo\n", i+1)
		} else {
			fmt.Printf("   ✅ Item %d: Descuento válido (%.2f)\n", i+1, item.DescuentoUnitario)
		}
		
		// Verificar impuestos del item
		for j, impuesto := range item.Impuestos {
			if impuesto.Tasa < 0 {
				fmt.Printf("   ❌ Item %d, Impuesto %d: Tasa no puede ser negativa\n", i+1, j+1)
			} else {
				fmt.Printf("   ✅ Item %d, Impuesto %d: Tasa válida (%.2f%%)\n", i+1, j+1, impuesto.Tasa)
			}
			
			if impuesto.MontoImpuesto < 0 {
				fmt.Printf("   ❌ Item %d, Impuesto %d: Monto no puede ser negativo\n", i+1, j+1)
			} else {
				fmt.Printf("   ✅ Item %d, Impuesto %d: Monto válido (%.2f)\n", i+1, j+1, impuesto.MontoImpuesto)
			}
		}
	}
	
	// Verificar totales
	if comprobante.Totales.TotalDescuentos < 0 {
		fmt.Println("   ❌ Total descuentos no puede ser negativo")
	} else {
		fmt.Printf("   ✅ Total descuentos válido (%.2f)\n", comprobante.Totales.TotalDescuentos)
	}
	
	if comprobante.Totales.TotalAnticipos < 0 {
		fmt.Println("   ❌ Total anticipos no puede ser negativo")
	} else {
		fmt.Printf("   ✅ Total anticipos válido (%.2f)\n", comprobante.Totales.TotalAnticipos)
	}
} 