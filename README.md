# 🧾 API de Facturación Electrónica SUNAT - Go

API profesional para la generación, firma digital y envío de comprobantes electrónicos a SUNAT (Perú) desarrollada en Go.

## 🎯 Características

- ✅ **Conversión automática a UBL 2.1** - Estándar SUNAT
- ✅ **Firma digital** con certificados PKCS#12
- ✅ **Empaquetado ZIP** según especificaciones SUNAT
- ✅ **Integración directa** con servicios SUNAT
- ✅ **Arquitectura modular** y extensible
- ✅ **Base de datos PostgreSQL** para persistencia
- ✅ **API REST** con 4 endpoints principales

## 🚀 Flujo de Trabajo SUNAT

```
1. Crear Comprobante → 2. Firmar y Empaquetar → 3. Enviar a SUNAT → 4. Obtener Resultado
```

### **Paso 1: Crear y Convertir a UBL**
```bash
POST /api/v1/comprobantes/
```
- Registra el comprobante en la base de datos
- Convierte automáticamente a formato UBL 2.1
- Retorna el XML UBL generado

### **Paso 2: Firmar y Empaquetar**
```bash
POST /api/v1/comprobantes/{id}/sign
```
- Firma digitalmente el XML UBL
- Empaqueta el XML firmado en ZIP
- Guarda tanto el XML firmado como el ZIP
- Retorna el XML firmado y ZIP en Base64

### **Paso 3: Enviar a SUNAT**
```bash
POST /api/v1/comprobantes/{id}/send
```
- Envía el ZIP empaquetado a SUNAT
- Obtiene y almacena el CDR
- Retorna el CDR y estado de SUNAT

### **Paso 4: Obtener Resultado**
```bash
GET /api/v1/comprobantes/{id}/result
```
- Retorna el estado completo
- Incluye XML firmado, ZIP y CDR
- Base64 para descarga

## 📋 Endpoints Disponibles

### **APIs Principales (Flujo SUNAT)**
| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/v1/comprobantes/` | Crear comprobante y generar XML UBL |
| POST | `/api/v1/comprobantes/{id}/sign` | Firmar XML UBL y empaquetar en ZIP |
| POST | `/api/v1/comprobantes/{id}/send` | Enviar ZIP a SUNAT y obtener CDR |
| GET | `/api/v1/comprobantes/{id}/result` | Consultar estado, XML firmado, ZIP y CDR |

### **APIs de Utilidades y Pruebas**
| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/v1/utils/generate-xml` | Generar XML UBL sin firma (para pruebas) |
| POST | `/api/v1/utils/convert-ubl` | Convertir comprobante a estructura UBL |
| POST | `/api/v1/utils/calculate-totals` | Calcular totales de un comprobante |

## 🚀 Endpoints Principales SUNAT

A partir de la versión actual, **el endpoint de creación de comprobante realiza automáticamente todo el flujo SUNAT**:
- Genera el comprobante en la base de datos
- Convierte a UBL 2.1 según el tipo (Factura, Boleta, Nota de Crédito, Nota de Débito)
- Serializa a XML
- Firma digitalmente el XML
- Empaqueta el XML firmado en ZIP
- Guarda el XML firmado y el ZIP en la carpeta `xml_pruebas/` y en la base de datos
- Retorna el comprobante, el XML UBL, el XML firmado y el ZIP (en base64)

### **1. Crear Comprobante (Factura, Boleta, Nota de Crédito, Nota de Débito)**
```http
POST /api/v1/comprobantes/
Content-Type: application/json
{
  "tipo": 1, // 1=Factura, 2=Boleta, 3=Nota de Crédito, 4=Nota de Débito
  "serie": "F001",
  "numero": "00000001",
  "fecha_emision": "2024-01-15T10:30:00Z",
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
      "impuesto_item": [
        {
          "tipo_impuesto": "1000",
          "codigo_impuesto": "IGV",
          "base_imponible": 200,
          "tasa": 18,
          "monto_impuesto": 36
        }
      ],
      "valor_venta": 200,
      "valor_total": 236
    }
  ]
}
```

**Respuesta:**
```json
{
  "message": "Comprobante creado, firmado y zipeado exitosamente",
  "id": "...",
  "xml_ubl": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>...",
  "xml_firmado": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>...",
  "zip_base64": "UEsDBBQAAAAIAAAAIQAA..."
}
```

- Para **Boleta**: usa `"tipo": 2` y la estructura es igual a factura.
- Para **Nota de Crédito**: usa `"tipo": 3` y sigue el estándar UBL de nota de crédito.
- Para **Nota de Débito**: usa `"tipo": 4` y sigue el estándar UBL de nota de débito.

### **2. Consultar comprobante por ID**
```http
GET /api/v1/comprobantes/{id}
```
**Respuesta:**
```json
{
  "comprobante": { ... }
}
```

### **3. Descargar XML firmado**
```http
GET /api/v1/comprobantes/{id}/xml?type=signed
```

### **4. Descargar ZIP firmado**
```http
GET /api/v1/comprobantes/{id}/zip
```

### **5. Enviar a SUNAT**
```http
POST /api/v1/comprobantes/{id}/send
```
**Respuesta:**
```json
{
  "message": "Enviado a SUNAT exitosamente",
  "cdr": "UEsDBBQAAAAIAAAAIQAA...",
  "status": 200
}
```

### **6. Consultar resultado SUNAT**
```http
GET /api/v1/comprobantes/{id}/result
```
**Respuesta:**
```json
{
  "estado": "ENVIADO",
  "xml_firmado": "...",
  "zip_base64": "...",
  "cdr_base64": "..."
}
```

---

**Catálogo de tipos de comprobante:**
- 1 = Factura
- 2 = Boleta
- 3 = Nota de Crédito
- 4 = Nota de Débito

**Todos los comprobantes generados cumplen el estándar SUNAT UBL 2.1, según los manuales y ejemplos oficiales:**
[Ver documentación y ejemplos SUNAT](https://drive.google.com/drive/folders/1JUh3RNS72pOIytIFevsrr_rADF_UuwHi)

## 🧪 Endpoints de Utilidades y Pruebas

### **Generar XML sin Firma (Para Pruebas)**
```http
POST /api/v1/utils/generate-xml
Content-Type: application/json
{
  "tipo": "01",
  "serie": "F001",
  "numero": "00000001",
  "fecha_emision": "2024-01-15T10:30:00Z",
  "tipo_moneda": "PEN",
  "emisor": {
    "ruc": "20123456789",
    "razon_social": "EMPRESA DE PRUEBA SAC",
    "nombre_comercial": "EMPRESA DE PRUEBA",
    "direccion": "AV. AREQUIPA 123",
    "distrito": "LIMA",
    "provincia": "LIMA",
    "departamento": "LIMA",
    "pais": "PE"
  },
  "receptor": {
    "tipo_documento": "1",
    "numero_documento": "12345678",
    "razon_social": "CLIENTE DE PRUEBA",
    "direccion": "AV. TEST 456",
    "distrito": "LIMA",
    "provincia": "LIMA",
    "departamento": "LIMA",
    "pais": "PE"
  },
  "items": [
    {
      "codigo": "PROD001",
      "descripcion": "PRODUCTO DE PRUEBA",
      "cantidad": 1,
      "precio_unitario": 100.00,
      "impuestos": [
        {
          "tipo": "1000",
          "porcentaje": 18.0
        }
      ]
    }
  ]
}
```

**Respuesta:**
```json
{
  "message": "XML generado sin firma exitosamente",
  "document_id": "20123456789-01-F001-00000001",
  "file_name": "20123456789-01-F001-00000001.xml",
  "xml_content": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>...",
  "ubl_structure": { ... },
  "totals": {
    "total_gravado": 100.00,
    "total_igv": 18.00,
    "total_payable": 118.00
  }
}
```

**Características del endpoint:**
- ✅ Genera XML UBL 2.1 sin firma digital
- ✅ Calcula totales automáticamente
- ✅ Guarda el archivo en `xml_pruebas/`
- ✅ Retorna el XML completo y estructura UBL
- ✅ Ideal para validar estructura antes de firmar
- ✅ Soporta todos los tipos de comprobantes (01, 03, 07, 08)

### **Convertir a UBL**
```http
POST /api/v1/utils/convert-ubl
Content-Type: application/json
{
  // Mismo formato que generate-xml
}
```

**Respuesta:**
```json
{
  "ubl_document": { ... },
  "xml": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>...",
  "totals": { ... }
}
```

### **Calcular Totales**
```http
POST /api/v1/utils/calculate-totals
Content-Type: application/json
{
  // Mismo formato que generate-xml
}
```

**Respuesta:**
```json
{
  "totals": { ... },
  "taxes": { ... },
  "items": { ... }
}
```

## 🧪 Pruebas Unitarias

### **Ejecutar Pruebas**
```bash
# Ejecutar todas las pruebas
go test ./...

# Ejecutar pruebas específicas
go test ./internal/handlers -v
go test ./internal/services -v

# Ejecutar pruebas con cobertura
go test ./... -cover
```

### **Pruebas Disponibles**

#### **Handlers (API Endpoints)**
- ✅ `TestGenerateXMLWithoutSignature` - Generación de XML sin firma
- ✅ `TestGenerateXMLWithoutSignatureInvalidData` - Validación de datos inválidos
- ✅ `TestGenerateXMLWithoutSignatureDifferentTypes` - Diferentes tipos de comprobantes

#### **Servicios SUNAT**
- ✅ `TestSUNATServiceSendDocumentAccepted` - Envío exitoso a SUNAT
- ✅ `TestSUNATServiceSendDocumentRejected` - Rechazo por SUNAT
- ✅ `TestSUNATServiceSendDocumentError` - Errores de comunicación
- ✅ `TestSUNATServiceBuildSOAPRequest` - Construcción de SOAP
- ✅ `TestSUNATServiceParseSOAPResponse` - Parsing de respuestas
- ✅ `TestSUNATServiceValidatePackage` - Validación de paquetes

### **Script de Prueba Manual**
```bash
# Ejecutar script de prueba del endpoint
go run scripts/test_xml_generation.go
```

## 🛠️ Instalación y Configuración

### **Prerrequisitos**
- Go 1.21+
- PostgreSQL 12+
- Certificado digital SUNAT (.p12)

### **1. Clonar Repositorio**
```bash
git clone <repository-url>
cd facturacion_sunat_api_go
```

### **2. Configurar Base de Datos**
```bash
# Crear base de datos PostgreSQL
createdb facturacion_sunat
```

### **3. Configurar Variables**
```bash
# Copiar archivo de configuración
cp config/app.yaml.example config/app.yaml

# Editar configuración
nano config/app.yaml
```

### **4. Configurar Certificados**
```bash
# Crear directorio de certificados
mkdir certs

# Copiar certificado SUNAT
cp tu_certificado.p12 certs/cert.p12
```

### **5. Instalar Dependencias**
```bash
go mod download
```

### **6. Ejecutar Servidor**
```bash
go run cmd/server/main.go
```

## ⚙️ Configuración

### **config/app.yaml**
```yaml
server:
  host: "0.0.0.0"
  port: "8080"
  read_timeout: 30
  write_timeout: 30

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  database: "facturacion_sunat"
  user: "postgres"
  password: "tu_password"
  ssl_mode: "disable"
  schema: "public"

sunat:
  base_url: "https://www.sunat.gob.pe/ol-ti-itcpfegem/billService"
  beta_url: "https://www.sunat.gob.pe/ol-ti-itcpfegem/billService"
  ruc: "20123456789"
  username: "tu_usuario"
  password: "tu_password"
  timeout: 60
  max_retries: 3

security:
  certificate_path: "./certs/cert.p12"
  certificate_pass: "tu_password"
  hash_algorithm: "SHA256"
```

## 📊 Monitoreo

### **Endpoints de Salud**
```bash
# Estado general
GET /health/

# Estado de base de datos
GET /health/database

# Estado de SUNAT
GET /health/sunat
```

## 🏗️ Arquitectura

```
facturacion_sunat_api_go/
├── cmd/server/main.go              # Punto de entrada
├── internal/
│   ├── config/                     # Configuración
│   ├── handlers/                   # Controladores HTTP
│   ├── models/                     # Modelos de datos
│   ├── repository/                 # Acceso a datos
│   ├── services/                   # Lógica de negocio
│   └── middleware/                 # Middleware HTTP
├── pkg/                           # Paquetes reutilizables
│   ├── certificate/               # Manejo de certificados
│   ├── encoding/                  # Codificación Base64/ZIP
│   └── sunat/                     # Cliente SUNAT
└── config/                        # Archivos de configuración
```

## 🔧 Servicios Principales

### **ConversionService**
- Convierte modelos de negocio a UBL 2.1
- Soporta Facturas, Boletas, Notas de Crédito/Débito

### **SigningService**
- Firma digital con certificados PKCS#12
- Genera XML canónico
- Aplica algoritmos SHA-256/RSA

### **EncodingService**
- Codificación Base64
- Creación de archivos ZIP
- Empaquetado para SUNAT

### **SUNATService**
- Cliente HTTP para servicios SUNAT
- Manejo de respuestas y errores
- Descarga de CDR

## 📝 Tipos de Comprobantes

| Tipo | Código | Descripción |
|------|--------|-------------|
| Factura | 1 | Factura electrónica |
| Boleta | 2 | Boleta de venta |
| Nota Crédito | 3 | Nota de crédito |
| Nota Débito | 4 | Nota de débito |

## 🔍 Estados del Proceso

| Estado | Código | Descripción |
|--------|--------|-------------|
| Pendiente | 1 | Comprobante creado |
| Procesando | 2 | En conversión UBL |
| Firmado | 3 | XML firmado |
| Enviado | 4 | Enviado a SUNAT |
| Aceptado | 5 | Aceptado por SUNAT |
| Rechazado | 6 | Rechazado por SUNAT |
| Error | 7 | Error en proceso |

## 🚨 Manejo de Errores

### **Códigos de Error Comunes**
- `400` - Datos inválidos
- `404` - Comprobante no encontrado
- `500` - Error interno del servidor

### **Ejemplo de Error**
```json
{
  "error": "Error firmando XML",
  "details": "certificado inválido"
}
```

## 🔒 Seguridad

- **Certificados PKCS#12** para firma digital
- **Validación de datos** en todos los endpoints
- **Logs de auditoría** para trazabilidad
- **Manejo seguro** de contraseñas

## 📈 Logs y Auditoría

El sistema registra automáticamente:
- Creación de comprobantes
- Procesos de firma
- Envíos a SUNAT
- Respuestas y errores
- Tiempos de procesamiento

## 🤝 Contribución

1. Fork el proyecto
2. Crea una rama para tu feature
3. Commit tus cambios
4. Push a la rama
5. Abre un Pull Request

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 📞 Soporte

Para soporte técnico o consultas:
- 📧 Email: soporte@tuempresa.com
- 📱 Teléfono: +51 1 123-4567
- 🌐 Web: https://tuempresa.com

---

**Desarrollado con ❤️ para la comunidad SUNAT** 

### Endpoints API FE Perú

| Método | Endpoint                | Descripción                        |
|--------|-------------------------|------------------------------------|
| GET    | /api/contribuyente      | Consulta de contribuyente por RUC  |
| GET    | /api/validar_ruc        | Valida un RUC                      |
| GET    | /api/soles              | Consulta tasa de cambio            |
| GET    | /api/calculadora        | Calculadora de cambio de moneda    |

Todos requieren el parámetro `apikey` en query o header. 

### Ejemplo de cuerpo JSON para POST /api/v1/invoices, /credit-notes, /debit-notes

**Request:**
```json
{
  "emisor": {
    "ruc": "20123456789",
    "razon_social": "EMPRESA DEMO S.A.C.",
    "certificado": "base64_cert",
    "clave_certificado": "clave123"
  },
  "receptor": {
    "tipo_documento": "6",
    "numero_documento": "10467793549",
    "nombre": "CLIENTE DEMO S.A.C."
  },
  "comprobante": {
    "tipo": 1, // 1=Factura, 3=Nota de Crédito, 4=Nota de Débito
    "serie": "F001",
    "numero": "00000001",
    "fecha": "2024-07-01T10:30:00Z",
    "moneda": "PEN"
  },
  "detalle": [
    {
      "codigo": "P001",
      "descripcion": "Producto de prueba",
      "unidad_medida": "NIU",
      "cantidad": 2,
      "precio_unitario": 100.00,
      "igv": 18.00
    }
  ],
  "totales": {
    "importe_total": 236.00,
    "total_igv": 36.00
  },
  "adicionales": {
    "guia_remision": "T001-123",
    "detraccion": false
  }
}
```

**Respuesta estándar:**
```json
{
  "estado": "aceptado",
  "hash": "HASH1234567890",
  "cdr_zip": "UEsDBBQAAAAIAAAAIQAA...",
  "xml_firmado": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbm...",
  "pdf_url": "https://tuservidor.com/pdfs/12345.pdf"
}
``` 