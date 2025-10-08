# Solución de Problemas: Error RUC Vacío

## Error: `cac:PartyIdentification/cbc:ID - No se encontro el ID-RUC o está vacío`

### Causas Posibles

1. **RUC vacío en el request JSON**
2. **Campo RUC no se está enviando correctamente**
3. **Problema en la validación del modelo**
4. **Error en la conversión a UBL**

### Pasos para Diagnosticar

#### 1. Verificar el Request JSON

Asegúrate de que el campo `ruc` esté presente y no vacío en el emisor:

```json
{
  "emisor": {
    "ruc": "20123456789",  // ← DEBE estar presente y no vacío
    "razon_social": "EMPRESA DEMO S.A.C.",
    // ... otros campos
  }
}
```

#### 2. Usar el Script de Prueba

Ejecuta el script de prueba que creamos:

```bash
cd scripts
go run test_create_comprobante.go
```

Este script:
- Envía un request válido con RUC correcto
- Muestra logs de debug
- Verifica que el RUC esté en el XML generado
- Confirma que el XML no tenga firma

#### 3. Verificar Logs del Servidor

El servidor ahora incluye logs de debug. Cuando hagas un request, verás en la consola:

```
DEBUG - Emisor RUC: '20123456789'
DEBUG - Emisor RazonSocial: 'EMPRESA DEMO S.A.C.'
DEBUG - Emisor TipoDocumento: '6'
```

Si ves `DEBUG - Emisor RUC: ''` (vacío), el problema está en el request.

#### 4. Validar con el Ejemplo Correcto

Usa el archivo `examples/valid_comprobante_request.json` como referencia:

```bash
curl -X POST http://localhost:8080/api/v1/comprobantes/ \
  -H "Content-Type: application/json" \
  -d @examples/valid_comprobante_request.json
```

### Soluciones

#### Solución 1: Verificar el Request

Asegúrate de que tu request incluya todos los campos requeridos del emisor:

```json
{
  "emisor": {
    "ruc": "20123456789",
    "razon_social": "EMPRESA DEMO S.A.C.",
    "nombre_comercial": "EMPRESA DEMO",
    "tipo_documento": "6",
    "direccion": "AV. AREQUIPA 123",
    "distrito": "LIMA",
    "provincia": "LIMA",
    "departamento": "LIMA",
    "codigo_pais": "PE"
  }
}
```

#### Solución 2: Reiniciar el Servidor

Si has hecho cambios en el código, reinicia el servidor:

```bash
# Detener el servidor (Ctrl+C)
# Luego ejecutar:
go run cmd/server/main.go
```

#### Solución 3: Verificar Validaciones

El código ahora incluye validaciones adicionales:

```go
// Validar que el RUC no esté vacío
if emisor.RUC == "" {
    return nil, fmt.Errorf("RUC emisor está vacío")
}

// Validar RUC según especificaciones SUNAT
if err := models.ValidarRUC(emisor.RUC); err != nil {
    return nil, fmt.Errorf("RUC emisor inválido: %v", err)
}
```

### Casos de Prueba

#### Caso 1: Request Válido
```json
{
  "tipo": 1,
  "serie": "F001",
  "numero": "00000001",
  "tipo_moneda": "PEN",
  "emisor": {
    "ruc": "20123456789",
    "razon_social": "EMPRESA DEMO S.A.C.",
    "tipo_documento": "6",
    "direccion": "AV. AREQUIPA 123",
    "distrito": "LIMA",
    "provincia": "LIMA",
    "departamento": "LIMA",
    "codigo_pais": "PE"
  },
  "receptor": {
    "tipo_documento": "6",
    "numero_documento": "12345678",
    "razon_social": "CLIENTE DEMO S.A.C."
  },
  "items": [
    {
      "id": 1,
      "numero_item": 1,
      "codigo": "P001",
      "descripcion": "Producto de prueba",
      "unidad_medida": "NIU",
      "cantidad": 2,
      "valor_unitario": 100,
      "precio_unitario": 118,
      "tipo_afectacion": 10,
      "valor_venta": 200,
      "valor_total": 236
    }
  ]
}
```

**Resultado esperado**: Status 200, XML generado sin firma, RUC presente en el XML.

#### Caso 2: RUC Vacío
```json
{
  "emisor": {
    "ruc": "",  // ← RUC vacío
    "razon_social": "EMPRESA DEMO S.A.C.",
    // ... otros campos
  }
}
```

**Resultado esperado**: Status 500, error "RUC emisor está vacío".

#### Caso 3: RUC Inválido
```json
{
  "emisor": {
    "ruc": "123",  // ← RUC inválido (muy corto)
    "razon_social": "EMPRESA DEMO S.A.C.",
    // ... otros campos
  }
}
```

**Resultado esperado**: Status 500, error "RUC emisor inválido".

### Verificación Final

Después de aplicar las correcciones:

1. **Reinicia el servidor**
2. **Ejecuta el script de prueba**: `go run scripts/test_create_comprobante.go`
3. **Verifica que el XML generado contenga el RUC**
4. **Confirma que el XML NO tenga firma**
5. **Revisa que el archivo se guarde en `xml_pruebas/`**

### Logs de Debug

El servidor ahora muestra logs detallados. Si sigues teniendo problemas, comparte:

1. Los logs de debug del servidor
2. El request JSON que estás enviando
3. La respuesta completa del servidor
4. El contenido del archivo XML generado en `xml_pruebas/` 