package models

import (
	"fmt"
	"time"
)

// Comprobante representa el modelo de negocio base
type Comprobante struct {
	ID                string                 `json:"id" db:"id"`
	Tipo              TipoComprobante        `json:"tipo" db:"tipo"`
	Serie             string                 `json:"serie" db:"serie"`
	Numero            string                 `json:"numero" db:"numero"`
	FechaEmision      time.Time              `json:"fecha_emision" db:"fecha_emision"`
	FechaVencimiento  *time.Time             `json:"fecha_vencimiento,omitempty" db:"fecha_vencimiento"`
	TipoMoneda        string                 `json:"tipo_moneda" db:"tipo_moneda"`
	Emisor            Emisor                 `json:"emisor"`
	Receptor          Receptor               `json:"receptor"`
	Items             []Item                 `json:"items"`
	Totales           Totales                `json:"totales"`
	Impuestos         []Impuesto             `json:"impuestos"`
	FormaPago         *FormaPago             `json:"forma_pago,omitempty"`
	Observaciones     string                 `json:"observaciones,omitempty" db:"observaciones"`
	EstadoProceso     EstadoProceso          `json:"estado_proceso" db:"estado_proceso"`
	XMLGenerado       string                 `json:"xml_generado,omitempty" db:"xml_generado"`
	XMLFirmado        string                 `json:"xml_firmado,omitempty" db:"xml_firmado"`
	ArchivoZIP        []byte                 `json:"archivo_zip,omitempty" db:"archivo_zip"`
	TicketSUNAT       string                 `json:"ticket_sunat,omitempty" db:"ticket_sunat"`
	CDRSUNAT          []byte                 `json:"cdr_sunat,omitempty" db:"cdr_sunat"`
	EstadoSUNAT       string                 `json:"estado_sunat,omitempty" db:"estado_sunat"`
	UsuarioCreacion   string                 `json:"usuario_creacion,omitempty" db:"usuario_creacion"`
	FechaCreacion     time.Time              `json:"fecha_creacion" db:"fecha_creacion"`
	FechaActualizacion time.Time             `json:"fecha_actualizacion" db:"fecha_actualizacion"`
}

type TipoComprobante int

const (
	TipoFactura TipoComprobante = iota + 1
	TipoBoleta
	TipoNotaCredito
	TipoNotaDebito
)

func (t TipoComprobante) String() string {
	switch t {
	case TipoFactura:
		return "01"
	case TipoBoleta:
		return "03"
	case TipoNotaCredito:
		return "07"
	case TipoNotaDebito:
		return "08"
	default:
		return "01"
	}
}

type EstadoProceso int

const (
	EstadoPendiente EstadoProceso = iota + 1
	EstadoProcesando
	EstadoFirmado
	EstadoEnviado
	EstadoAceptado
	EstadoRechazado
	EstadoError
)

type Emisor struct {
	RUC                string `json:"ruc" validate:"required,len=11"`
	RazonSocial        string `json:"razon_social" validate:"required"`
	NombreComercial    string `json:"nombre_comercial,omitempty"`
	TipoDocumento      string `json:"tipo_documento" validate:"required"`
	Direccion          string `json:"direccion" validate:"required"`
	Distrito           string `json:"distrito" validate:"required"`
	Provincia          string `json:"provincia" validate:"required"`
	Departamento       string `json:"departamento" validate:"required"`
	CodigoPostal       string `json:"codigo_postal,omitempty"`
	CodigoPais         string `json:"codigo_pais" validate:"required"`
	Telefono           string `json:"telefono,omitempty"`
	Email              string `json:"email,omitempty"`
}

type Receptor struct {
	TipoDocumento   string `json:"tipo_documento" validate:"required"`
	NumeroDocumento string `json:"numero_documento" validate:"required"`
	RazonSocial     string `json:"razon_social" validate:"required"`
	Direccion       string `json:"direccion,omitempty"`
	Email           string `json:"email,omitempty"`
}

type Item struct {
	ID                  int                 `json:"id"`
	NumeroItem          int                 `json:"numero_item" validate:"required,gt=0"`
	Codigo              string              `json:"codigo" validate:"required"`
	CodigoSUNAT         string              `json:"codigo_sunat,omitempty"`
	Descripcion         string              `json:"descripcion" validate:"required"`
	UnidadMedida        string              `json:"unidad_medida" validate:"required"`
	Cantidad            float64             `json:"cantidad" validate:"required,gt=0"`
	ValorUnitario       float64             `json:"valor_unitario" validate:"required,gte=0"`
	PrecioUnitario      float64             `json:"precio_unitario" validate:"required,gte=0"`
	DescuentoUnitario   float64             `json:"descuento_unitario,omitempty"`
	TipoAfectacion      TipoAfectacionIGV   `json:"tipo_afectacion" validate:"required"`
	ImpuestoItem        []ImpuestoItem      `json:"impuesto_item,omitempty"`
	ValorVenta          float64             `json:"valor_venta"`
	ValorTotal          float64             `json:"valor_total"`
}

// TipoAfectacionIGV según especificaciones SUNAT
type TipoAfectacionIGV int

const (
	GravadoOneroso TipoAfectacionIGV = iota + 10
	GravadoGratuito
	Exonerado
	Inafecto
	ExportacionOperacionGratuita
	GravadoIVAP
	ExoneradoIVAP
	InafectoIVAP
)

func (t TipoAfectacionIGV) String() string {
	switch t {
	case GravadoOneroso:
		return "10" // Gravado - Operación Onerosa
	case GravadoGratuito:
		return "11" // Gravado - Operación Gratuita
	case Exonerado:
		return "20" // Exonerado - Operación Onerosa
	case Inafecto:
		return "30" // Inafecto - Operación Onerosa
	case ExportacionOperacionGratuita:
		return "40" // Exportación - Operación Gratuita
	case GravadoIVAP:
		return "17" // Gravado IVAP
	case ExoneradoIVAP:
		return "27" // Exonerado IVAP
	case InafectoIVAP:
		return "37" // Inafecto IVAP
	default:
		return "10"
	}
}

// ValidarRUC valida formato de RUC según especificaciones SUNAT
func ValidarRUC(ruc string) error {
	if len(ruc) != 11 {
		return fmt.Errorf("RUC debe tener 11 dígitos")
	}
	
	// Validar que sean solo números
	for _, char := range ruc {
		if char < '0' || char > '9' {
			return fmt.Errorf("RUC debe contener solo números")
		}
	}
	
	// Validar tipo de contribuyente (primeros 2 dígitos)
	tipoContribuyente := ruc[:2]
	tiposValidos := []string{"10", "15", "17", "20"}
	valido := false
	for _, tipo := range tiposValidos {
		if tipoContribuyente == tipo {
			valido = true
			break
		}
	}
	
	if !valido {
		return fmt.Errorf("tipo de contribuyente inválido: %s", tipoContribuyente)
	}
	
	return nil
}

type ImpuestoItem struct {
	TipoImpuesto     string  `json:"tipo_impuesto" validate:"required"`
	CodigoImpuesto   string  `json:"codigo_impuesto" validate:"required"`
	BaseImponible    float64 `json:"base_imponible" validate:"required,gte=0"`
	Tasa             float64 `json:"tasa" validate:"required,gte=0"`
	MontoImpuesto    float64 `json:"monto_impuesto" validate:"required,gte=0"`
}

type Totales struct {
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

type Impuesto struct {
	TipoImpuesto      string  `json:"tipo_impuesto" validate:"required"`
	CodigoImpuesto    string  `json:"codigo_impuesto" validate:"required"`
	BaseImponible     float64 `json:"base_imponible" validate:"required,gte=0"`
	Tasa              float64 `json:"tasa" validate:"required,gte=0"`
	MontoImpuesto     float64 `json:"monto_impuesto" validate:"required,gte=0"`
}

type FormaPago struct {
	TipoPago     string      `json:"tipo_pago" validate:"required"`
	MontoContado float64     `json:"monto_contado,omitempty"`
	Cuotas       []Cuota     `json:"cuotas,omitempty"`
}

type Cuota struct {
	NumeroCuota    int       `json:"numero_cuota" validate:"required,gt=0"`
	FechaVencimiento time.Time `json:"fecha_vencimiento" validate:"required"`
	Monto          float64   `json:"monto" validate:"required,gt=0"`
}

// Factura específica
type Factura struct {
	Comprobante
	DetraccionAplicada bool    `json:"detraccion_aplicada"`
	MontoDetraccion    float64 `json:"monto_detraccion,omitempty"`
	PorcentajeDetraccion float64 `json:"porcentaje_detraccion,omitempty"`
	OrdenCompra        string  `json:"orden_compra,omitempty"`
	GuiaRemision       string  `json:"guia_remision,omitempty"`
}

// Boleta específica
type Boleta struct {
	Comprobante
}

// Lote representa un lote de comprobantes para procesamiento
type Lote struct {
	ID                   string                 `json:"id"`
	Descripcion          string                 `json:"descripcion"`
	TotalDocumentos      int                    `json:"total_documentos"`
	DocumentosProcesados int                    `json:"documentos_procesados"`
	DocumentosExitosos   int                    `json:"documentos_exitosos"`
	DocumentosFallidos   int                    `json:"documentos_fallidos"`
	Estado               string                 `json:"estado"`
	FechaInicio          *time.Time             `json:"fecha_inicio"`
	FechaFin             *time.Time             `json:"fecha_fin"`
	ConfiguracionProceso map[string]interface{} `json:"configuracion_proceso"`
	FechaCreacion        time.Time              `json:"fecha_creacion"`
	FechaActualizacion   time.Time              `json:"fecha_actualizacion"`
	UsuarioCreacion      string                 `json:"usuario_creacion"`
}