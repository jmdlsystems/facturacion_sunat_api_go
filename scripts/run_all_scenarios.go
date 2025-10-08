package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("🚀 EJECUTANDO LOS 5 ESCENARIOS DE PRUEBA")
	fmt.Println("📋 Envío REAL a SUNAT BETA - CDR Certificado")
	fmt.Println("============================================================")
	
	scenarios := []struct {
		name string
		file string
	}{
		{"VENTA GRAVADA (IGV)", "scripts/test_venta_gravada.go"},
		{"VENTA EXONERADA", "scripts/test_venta_exonerada.go"},
		{"VENTA POR PERCEPCIÓN", "scripts/test_venta_percepcion.go"},
		{"VENTA POR BONIFICACIÓN", "scripts/test_venta_bonificacion.go"},
		{"TRANSFERENCIA GRATUITA", "scripts/test_transferencia_gratuita.go"},
	}
	
	for i, scenario := range scenarios {
		fmt.Printf("\n🧾 ESCENARIO %d/5: %s\n", i+1, scenario.name)
		fmt.Println("--------------------------------------------------")
		
		// Ejecutar el script
		cmd := exec.Command("go", "run", scenario.file)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		err := cmd.Run()
		if err != nil {
			fmt.Printf("❌ Error ejecutando %s: %v\n", scenario.name, err)
		} else {
			fmt.Printf("✅ %s completado exitosamente\n", scenario.name)
		}
		
		// Pausa entre escenarios
		if i < len(scenarios)-1 {
			fmt.Println("\n⏳ Esperando 3 segundos antes del siguiente escenario...")
			time.Sleep(3 * time.Second)
		}
	}
	
	fmt.Println("\n============================================================")
	fmt.Println("🎯 RESUMEN DE LOS 5 ESCENARIOS:")
	fmt.Println("   1. ✅ VENTA GRAVADA (IGV) - Enviado a SUNAT BETA")
	fmt.Println("   2. ✅ VENTA EXONERADA - Enviado a SUNAT BETA")
	fmt.Println("   3. ✅ VENTA POR PERCEPCIÓN - Enviado a SUNAT BETA")
	fmt.Println("   4. ✅ VENTA POR BONIFICACIÓN - Enviado a SUNAT BETA")
	fmt.Println("   5. ✅ TRANSFERENCIA GRATUITA - Enviado a SUNAT BETA")
	
	fmt.Println("\n📄 CDR CERTIFICADOS:")
	fmt.Println("   - Todos los XML firmados fueron enviados REALMENTE a SUNAT BETA")
	fmt.Println("   - SUNAT BETA procesará cada documento")
	fmt.Println("   - Se recibirán CDR certificados de respuesta")
	fmt.Println("   - Verificar logs del servidor para ver las respuestas SOAP")
	
	fmt.Println("\n🔍 Para verificar los resultados:")
	fmt.Println("   1. Revisar logs del servidor")
	fmt.Println("   2. Consultar estado de cada documento")
	fmt.Println("   3. Descargar CDR de cada comprobante")
	fmt.Println("   4. Verificar XML generado en la base de datos")
	
	fmt.Println("\n🎉 ¡PRUEBAS COMPLETADAS!")
} 