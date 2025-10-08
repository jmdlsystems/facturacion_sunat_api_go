package handlers

import (
	"facturacion_sunat_api_go/internal/models"
	"facturacion_sunat_api_go/internal/repository"
	"facturacion_sunat_api_go/internal/services"
	"net/http"
	"strconv"
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"facturacion_sunat_api_go/internal/config"
)

type ComprobanteHandler struct {
	repository        *repository.ComprobanteRepository
	conversionService *services.ConversionService
	signingService    *services.SigningService
	encodingService   *services.EncodingService
	sunatService      *services.SUNATService
}

func NewComprobanteHandler(
	repo *repository.ComprobanteRepository,
	conversionService *services.ConversionService,
	signingService *services.SigningService,
	encodingService *services.EncodingService,
	sunatService *services.SUNATService,
) *ComprobanteHandler {
	return &ComprobanteHandler{
		repository:        repo,
		conversionService: conversionService,
		signingService:    signingService,
		encodingService:   encodingService,
		sunatService:      sunatService,
	}
}

func saveToXMLPruebas(fileName string, data []byte) error {
	dir := "xml_pruebas"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fileName), data, 0644)
}

// CreateComprobante crea un nuevo comprobante, genera el XML UBL sin firma y lo guarda en xml_pruebas
func (h *ComprobanteHandler) CreateComprobante(c *gin.Context) {
	var comprobante models.Comprobante

	if err := c.ShouldBindJSON(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validación reforzada del RUC antes de procesar
	ruc := strings.TrimSpace(comprobante.Emisor.RUC)
	if len(ruc) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "RUC emisor inválido: debe tener 11 dígitos",
			"details": "El campo emisor.ruc debe tener exactamente 11 dígitos.",
		})
		return
	}
	for _, cRune := range []rune(ruc) {
		if cRune < '0' || cRune > '9' {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "RUC emisor inválido: solo debe contener números",
				"details": "El campo emisor.ruc solo debe contener números.",
			})
			return
		}
	}
	if err := models.ValidarRUC(ruc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "RUC emisor inválido",
			"details": err.Error(),
		})
		return
	}
	comprobante.Emisor.RUC = ruc // Asegura que el RUC limpio se use en todo el flujo

	// Validar razón social y dirección del emisor
	if comprobante.Emisor.RazonSocial == "" || comprobante.Emisor.Direccion == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos del emisor incompletos",
			"details": "Razón social y dirección del emisor son obligatorios.",
		})
		return
	}

	// Validar receptor según catálogo SUNAT
	receptor := comprobante.Receptor
	if receptor.TipoDocumento == "" || receptor.NumeroDocumento == "" || receptor.RazonSocial == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos del receptor incompletos",
			"details": "Tipo de documento, número y razón social del receptor son obligatorios.",
		})
		return
	}
	if receptor.TipoDocumento == "6" && len(receptor.NumeroDocumento) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "RUC receptor inválido",
			"details": "El receptor es empresa y debe tener RUC de 11 dígitos.",
		})
		return
	}
	if receptor.TipoDocumento == "1" && len(receptor.NumeroDocumento) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "DNI receptor inválido",
			"details": "El receptor es persona natural y debe tener DNI de 8 dígitos.",
		})
		return
	}

	// Validar fecha de emisión
	if comprobante.FechaEmision.IsZero() || comprobante.FechaEmision.Year() < 2000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Fecha de emisión inválida",
			"details": "La fecha de emisión debe ser válida y mayor al año 2000.",
		})
		return
	}

	// Validar tipo de comprobante y moneda según catálogo SUNAT
	if comprobante.Tipo.String() != "01" && comprobante.Tipo.String() != "03" && comprobante.Tipo.String() != "07" && comprobante.Tipo.String() != "08" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Tipo de comprobante inválido",
			"details": "Tipo debe ser 01, 03, 07 u 08 según catálogo SUNAT.",
		})
		return
	}
	if comprobante.TipoMoneda != "PEN" && comprobante.TipoMoneda != "USD" && comprobante.TipoMoneda != "EUR" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Moneda inválida",
			"details": "Solo se permite PEN, USD o EUR.",
		})
		return
	}

	// Validar que haya al menos un ítem
	if len(comprobante.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Comprobante sin ítems",
			"details": "Debe haber al menos un ítem en el comprobante.",
		})
		return
	}

	// Validar totales (simple)
	if comprobante.Totales.ImporteTotal <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Importe total inválido",
			"details": "El importe total debe ser mayor a cero.",
		})
		return
	}

	// Debug: Log de los datos del emisor y receptor
	fmt.Printf("DEBUG - Emisor RUC: '%s'\n", comprobante.Emisor.RUC)
	fmt.Printf("DEBUG - Emisor RazonSocial: '%s'\n", comprobante.Emisor.RazonSocial)
	fmt.Printf("DEBUG - Emisor TipoDocumento: '%s'\n", comprobante.Emisor.TipoDocumento)
	fmt.Printf("DEBUG - Receptor TipoDocumento: '%s'\n", comprobante.Receptor.TipoDocumento)
	fmt.Printf("DEBUG - Receptor NumeroDocumento: '%s'\n", comprobante.Receptor.NumeroDocumento)

	// Generar ID único
	comprobante.ID = uuid.New().String()
	comprobante.FechaCreacion = time.Now()
	comprobante.FechaActualizacion = time.Now()
	comprobante.EstadoProceso = models.EstadoPendiente

	// Calcular totales automáticamente
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error calculando totales",
			"details": err.Error(),
		})
		return
	}

	// Guardar en base de datos (estado pendiente)
	if err := h.repository.Create(&comprobante); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando comprobante",
			"details": err.Error(),
		})
		return
	}

	// 1. Generar estructura UBL
	ublStruct, err := h.conversionService.ConvertToUBL(&comprobante)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error generando estructura UBL",
			"details": err.Error(),
		})
		return
	}
	// 2. Serializar a XML
	xmlUBL, err := h.conversionService.UBLService.SerializeToXML(ublStruct)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error serializando UBL a XML",
			"details": err.Error(),
		})
		return
	}
	// 2.1 Validar que el XML generado sea bien formado
	if !isXMLWellFormed(xmlUBL) {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "El XML UBL generado no es bien formado",
			"details": "El XML generado no cumple la estructura XML estándar.",
		})
		return
	}

	// 3. Firmar el XML UBL
	certPath := config.AppConfig.Security.CertificatePath
	certPass := config.AppConfig.Security.CertificatePass
	xmlFirmado, err := h.signingService.SignDocument(xmlUBL, certPath, certPass)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error firmando XML",
			"details": err.Error(),
		})
		return
	}
	// 3.1 Validar que el XML firmado sea bien formado
	if !isXMLWellFormed(xmlFirmado) {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "El XML firmado no es bien formado",
			"details": "El XML firmado no cumple la estructura XML estándar.",
		})
		return
	}

	// 4. Empaquetar en ZIP
	documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero
	zipPkg, err := h.encodingService.ProcessForSUNAT(xmlFirmado, documentID)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error empaquetando ZIP",
			"details": err.Error(),
		})
		return
	}

	// 5. Guardar XML firmado y ZIP en la base de datos
	if err := h.repository.UpdateSignedXML(comprobante.ID, string(xmlFirmado)); err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando XML firmado",
			"details": err.Error(),
		})
		return
	}
	if err := h.repository.UpdateArchivoZIP(comprobante.ID, zipPkg.ZipContent); err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando ZIP",
			"details": err.Error(),
		})
		return
	}

	// 6. Guardar archivos en xml_pruebas
	fileNameXML := documentID + "-signed.xml"
	fileNameZIP := documentID + ".zip"
	_ = saveToXMLPruebas(fileNameXML, xmlFirmado)
	_ = saveToXMLPruebas(fileNameZIP, zipPkg.ZipContent)

	// 7. Responder con el comprobante, XML firmado y ZIP (en base64)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Comprobante creado, firmado y zipeado exitosamente",
		"id":          comprobante.ID,
		"xml_firmado": string(xmlFirmado),
		"zip_base64":  zipPkg.Base64Content,
		"file_xml":    fileNameXML,
		"file_zip":    fileNameZIP,
	})
}

// isXMLWellFormed valida que el XML sea bien formado
func isXMLWellFormed(xmlData []byte) bool {
	var v interface{}
	return xml.Unmarshal(xmlData, &v) == nil
}

// GetComprobante obtiene un comprobante por ID
func (h *ComprobanteHandler) GetComprobante(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comprobante": comprobante,
	})
}

// ListComprobantes lista comprobantes con paginación
func (h *ComprobanteHandler) ListComprobantes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	estado := c.Query("estado")
	tipo := c.Query("tipo")

	filters := map[string]interface{}{}
	if estado != "" {
		estadoInt, err := strconv.Atoi(estado)
		if err == nil {
			filters["estado_proceso"] = estadoInt
		}
	}
	if tipo != "" {
		tipoInt, err := strconv.Atoi(tipo)
		if err == nil {
			filters["tipo"] = tipoInt
		}
	}

	comprobantes, total, err := h.repository.List(page, limit, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error obteniendo comprobantes",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comprobantes": comprobantes,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// UpdateComprobante actualiza un comprobante
func (h *ComprobanteHandler) UpdateComprobante(c *gin.Context) {
	id := c.Param("id")

	// Verificar que existe
	existing, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	// Verificar que no esté procesado
	if existing.EstadoProceso != models.EstadoPendiente {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No se puede modificar un comprobante ya procesado",
		})
		return
	}

	var comprobante models.Comprobante
	if err := c.ShouldBindJSON(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	comprobante.ID = id
	comprobante.FechaActualizacion = time.Now()

	// Recalcular totales
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error calculando totales",
			"details": err.Error(),
		})
		return
	}

	if err := h.repository.Update(&comprobante); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error actualizando comprobante",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Comprobante actualizado exitosamente",
		"comprobante": comprobante,
	})
}

// DeleteComprobante elimina un comprobante
func (h *ComprobanteHandler) DeleteComprobante(c *gin.Context) {
	id := c.Param("id")

	// Verificar que existe
	existing, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	// Verificar que no esté enviado a SUNAT
	if existing.EstadoProceso == models.EstadoEnviado || existing.EstadoProceso == models.EstadoAceptado {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No se puede eliminar un comprobante enviado a SUNAT",
		})
		return
	}

	if err := h.repository.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error eliminando comprobante",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Comprobante eliminado exitosamente",
	})
}

// ProcessComprobante procesa un comprobante (conversión UBL + firmado)
func (h *ComprobanteHandler) ProcessComprobante(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	// Verificar estado
	if comprobante.EstadoProceso != models.EstadoPendiente {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El comprobante ya fue procesado",
		})
		return
	}

	// Actualizar estado a procesando
	comprobante.EstadoProceso = models.EstadoProcesando
	if err := h.repository.UpdateStatus(id, comprobante.EstadoProceso); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error actualizando estado",
		})
		return
	}

	// Convertir a UBL
	ublDocument, err := h.conversionService.ConvertToUBL(comprobante)
	if err != nil {
		h.repository.UpdateStatus(id, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error convirtiendo a UBL",
			"details": err.Error(),
		})
		return
	}

	// Serializar a XML
	xmlData, err := h.conversionService.UBLService.SerializeToXML(ublDocument)
	if err != nil {
		h.repository.UpdateStatus(id, models.EstadoError)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error serializando XML",
			"details": err.Error(),
		})
		return
	}

	// Guardar XML generado
	comprobante.EstadoProceso = models.EstadoFirmado
	if err := h.repository.UpdateXML(id, string(xmlData)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error guardando XML",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Comprobante procesado exitosamente",
		"xml_generated": true,
	})
}

// SignComprobante firma el XML UBL del comprobante, lo empaqueta en ZIP y lo guarda
func (h *ComprobanteHandler) SignComprobante(c *gin.Context) {
	id := c.Param("id")

	var request struct {
		CertificatePath string `json:"certificate_path"`
		CertificatePass string `json:"certificate_password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	if comprobante.XMLGenerado == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No hay XML UBL generado para firmar",
		})
		return
	}

	certPath := request.CertificatePath
	if certPath == "" {
		certPath = "./certs/cert.b64"
	}
	certPass := request.CertificatePass

	// 1. Firmar el XML UBL
	xmlFirmado, err := h.signingService.SignDocument([]byte(comprobante.XMLGenerado), certPath, certPass)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error firmando XML",
			"details": err.Error(),
		})
		return
	}

	// 2. Empaquetar en ZIP
	documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero
	zipPkg, err := h.encodingService.ProcessForSUNAT(xmlFirmado, documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error empaquetando ZIP",
			"details": err.Error(),
		})
		return
	}

	// 3. Guardar el XML firmado en la base de datos
	if err := h.repository.UpdateSignedXML(id, string(xmlFirmado)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando XML firmado",
			"details": err.Error(),
		})
		return
	}

	// 4. Guardar el ZIP en la base de datos
	if err := h.repository.UpdateArchivoZIP(id, zipPkg.ZipContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando ZIP",
			"details": err.Error(),
		})
		return
	}

	documentID = comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero
	fileNameXML := documentID + "-signed.xml"
	fileNameZIP := documentID + ".zip"
	_ = saveToXMLPruebas(fileNameXML, xmlFirmado)
	_ = saveToXMLPruebas(fileNameZIP, zipPkg.ZipContent)

	c.JSON(http.StatusOK, gin.H{
		"message":     "XML firmado y empaquetado exitosamente",
		"xml_firmado": string(xmlFirmado),
		"zip_base64":  zipPkg.Base64Content,
	})
}


// SendToSUNAT envía el ZIP empaquetado a SUNAT y obtiene el CDR
func (h *ComprobanteHandler) SendToSUNAT(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	if comprobante.XMLFirmado == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No hay XML firmado. Debe firmar el comprobante primero",
		})
		return
	}

	// Crear documentID una sola vez
	documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero

	// Verificar si existe el archivo ZIP, si no, crearlo
	var zipContent []byte
	if len(comprobante.ArchivoZIP) == 0 {
		fmt.Printf("⚠️  ZIP no encontrado en BD, creando desde XML firmado...\n")
		
		// Crear ZIP desde el XML firmado
		zipPkg, err := h.encodingService.ProcessForSUNAT([]byte(comprobante.XMLFirmado), documentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error creando archivo ZIP",
				"details": err.Error(),
			})
			return
		}
		
		// Validar que el ZIP se creó correctamente
		if len(zipPkg.ZipContent) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error: ZIP generado está vacío",
			})
			return
		}
		
		zipContent = zipPkg.ZipContent
		fmt.Printf("✅ ZIP creado: %d bytes\n", len(zipContent))
		
		// Guardar el ZIP en la base de datos
		if err := h.repository.UpdateArchivoZIP(id, zipContent); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error guardando archivo ZIP",
				"details": err.Error(),
			})
			return
		}
		fmt.Printf("✅ ZIP guardado en BD\n")
	} else {
		zipContent = comprobante.ArchivoZIP
		fmt.Printf("✅ ZIP encontrado en BD: %d bytes\n", len(zipContent))
	}

	// Validar que tenemos contenido ZIP válido
	if len(zipContent) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error: contenido ZIP vacío después de creación/recuperación",
		})
		return
	}

	// Crear paquete SUNAT con el ZIP
	zipPkg := &services.SUNATPackage{
		FileName:      documentID + ".zip",
		ZipContent:    zipContent,
		Base64Content: h.encodingService.EncodeToBase64(zipContent),
		XMLContent:    []byte(comprobante.XMLFirmado),
	}

	fmt.Printf("📦 Paquete SUNAT creado:\n")
	fmt.Printf("   - Nombre: %s\n", zipPkg.FileName)
	fmt.Printf("   - ZIP: %d bytes\n", len(zipPkg.ZipContent))
	fmt.Printf("   - XML: %d bytes\n", len(zipPkg.XMLContent))
	fmt.Printf("   - Base64: %d caracteres\n", len(zipPkg.Base64Content))

	// Validar el paquete antes de enviar
	if err := zipPkg.ValidatePackage(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Paquete inválido para envío a SUNAT",
			"details": err.Error(),
		})
		return
	}

	// Enviar a SUNAT
	response, err := h.sunatService.SendDocument(zipPkg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando a SUNAT",
			"details": err.Error(),
		})
		return
	}

	// Verificar si el envío fue exitoso
	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error en respuesta de SUNAT",
			"details": response.Message,
			"status":  response.StatusCode,
		})
		return
	}

	// Guardar el CDR en la base de datos
	if err := h.repository.UpdateCDR(id, response.ApplicationResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error guardando CDR",
			"details": err.Error(),
		})
		return
	}

	// Guardar CDR en archivo de prueba si existe
	if response.ApplicationResponse != nil && len(response.ApplicationResponse) > 0 {
		fileNameCDR := documentID + "-cdr.zip"
		_ = saveToXMLPruebas(fileNameCDR, response.ApplicationResponse)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Enviado a SUNAT exitosamente",
		"cdr":     response.ApplicationResponse,
		"status":  response.StatusCode,
	})
}

// GetSUNATStatus consulta el estado en SUNAT
func (h *ComprobanteHandler) GetSUNATStatus(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	if comprobante.TicketSUNAT == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El comprobante no ha sido enviado a SUNAT",
		})
		return
	}

	// Consultar estado en SUNAT
	statusResponse, err := h.sunatService.GetDocumentStatus(comprobante.TicketSUNAT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error consultando estado SUNAT",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket":      comprobante.TicketSUNAT,
		"status":      statusResponse.Status,
		"status_code": statusResponse.StatusCode,
		"description": statusResponse.Description,
		"timestamp":   statusResponse.Timestamp,
	})
}

// DownloadCDR descarga el CDR de SUNAT
func (h *ComprobanteHandler) DownloadCDR(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	if comprobante.TicketSUNAT == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El comprobante no ha sido enviado a SUNAT",
		})
		return
	}

	// Descargar CDR
	cdrResponse, err := h.sunatService.DownloadCDR(comprobante.Emisor.RUC, comprobante.Tipo.String(), comprobante.Serie, comprobante.Numero)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error descargando CDR",
			"details": err.Error(),
		})
		return
	}

	// Guardar CDR en base de datos
	if err := h.repository.UpdateCDR(id, cdrResponse.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error guardando CDR",
		})
		return
	}

	// Configurar headers para descarga
	fileName := comprobante.Serie + "-" + comprobante.Numero + "-CDR.zip"
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/zip")
	c.Data(http.StatusOK, "application/zip", cdrResponse.Content)
}

// DownloadXML descarga el XML del comprobante
func (h *ComprobanteHandler) DownloadXML(c *gin.Context) {
	id := c.Param("id")
	xmlType := c.DefaultQuery("type", "signed") // "original" o "signed"

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	var xmlData string
	var fileName string

	if xmlType == "signed" && comprobante.XMLFirmado != "" {
		xmlData = comprobante.XMLFirmado
		fileName = comprobante.Serie + "-" + comprobante.Numero + "-signed.xml"
	} else if comprobante.XMLGenerado != "" {
		xmlData = comprobante.XMLGenerado
		fileName = comprobante.Serie + "-" + comprobante.Numero + ".xml"
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "XML no disponible",
		})
		return
	}

	// Configurar headers para descarga
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/xml")
	c.Data(http.StatusOK, "application/xml", []byte(xmlData))
}

// Métodos adicionales para utilidades y validaciones

// ValidateRUC valida un RUC contra SUNAT y responde con información extendida
func (h *ComprobanteHandler) ValidateRUC(c *gin.Context) {
	ruc := c.Query("ruc")
	if len(ruc) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro ruc requerido y debe tener 11 dígitos"})
		return
	}
	// Primero validación local básica
	if err := models.ValidarRUC(ruc); err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}
	// Intentar validar contra SUNAT
	resp, err := h.sunatService.ValidateRUC(ruc)
	if err != nil {
		// Si falla la consulta a SUNAT, responde solo validación local
		c.JSON(http.StatusOK, gin.H{"valid": true, "warning": "Validación SUNAT no disponible", "ruc": ruc})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"valid":       resp.Valid,
		"ruc":         ruc,
		"razon_social": resp.RazonSocial,
		"estado":       resp.Estado,
		"direccion":    resp.Direccion,
		"timestamp":    resp.Timestamp,
	})
}

// ValidateCertificate valida un certificado
func (h *ComprobanteHandler) ValidateCertificate(c *gin.Context) {
	var request struct {
		CertificatePath string `json:"certificate_path" binding:"required"`
		CertificatePass string `json:"certificate_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar certificado
	if err := h.signingService.ValidateCertificate(request.CertificatePath, request.CertificatePass); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Obtener información del certificado
	info, err := h.signingService.GetCertificateInfo(request.CertificatePath, request.CertificatePass)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error obteniendo información del certificado",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":             true,
		"certificate_info": info,
	})
}

// ConvertToUBL convierte un comprobante a UBL sin guardarlo
func (h *ComprobanteHandler) ConvertToUBL(c *gin.Context) {
	var comprobante models.Comprobante

	if err := c.ShouldBindJSON(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Calcular totales
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error calculando totales",
			"details": err.Error(),
		})
		return
	}

	// Convertir a UBL
	ublDocument, err := h.conversionService.ConvertToUBL(&comprobante)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error convirtiendo a UBL",
			"details": err.Error(),
		})
		return
	}

	// Serializar a XML
	xmlData, err := h.conversionService.UBLService.SerializeToXML(ublDocument)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error serializando XML",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ubl_document": ublDocument,
		"xml":          string(xmlData),
		"totals":       comprobante.Totales,
	})
}

// GenerateXMLWithoutSignature genera XML UBL sin firma digital
func (h *ComprobanteHandler) GenerateXMLWithoutSignature(c *gin.Context) {
	var comprobante models.Comprobante

	if err := c.ShouldBindJSON(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Calcular totales automáticamente
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error calculando totales",
			"details": err.Error(),
		})
		return
	}

	// 1. Generar estructura UBL
	ublStruct, err := h.conversionService.ConvertToUBL(&comprobante)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error generando estructura UBL",
			"details": err.Error(),
		})
		return
	}

	// 2. Serializar a XML sin firma
	xmlUBL, err := h.conversionService.UBLService.SerializeToXML(ublStruct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error serializando UBL a XML",
			"details": err.Error(),
		})
		return
	}

	// 3. Formatear XML para SUNAT (sin firma)
	xmlFormateado, err := h.conversionService.UBLService.FormatXMLForSUNAT(xmlUBL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error formateando XML",
			"details": err.Error(),
		})
		return
	}

	// 4. Generar nombre de archivo
	documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero
	fileNameXML := documentID + ".xml"

	// 5. Guardar archivo en carpeta xml_pruebas
	_ = saveToXMLPruebas(fileNameXML, xmlFormateado)

	c.JSON(http.StatusOK, gin.H{
		"message":      "XML generado sin firma exitosamente",
		"document_id":  documentID,
		"file_name":    fileNameXML,
		"xml_content":  string(xmlFormateado),
		"ubl_structure": ublStruct,
		"totals":       comprobante.Totales,
	})
}

// CalculateTotals calcula totales de un comprobante
func (h *ComprobanteHandler) CalculateTotals(c *gin.Context) {
	var comprobante models.Comprobante

	if err := c.ShouldBindJSON(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Calcular totales
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Error calculando totales",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totals":   comprobante.Totales,
		"taxes":    comprobante.Impuestos,
		"items":    comprobante.Items,
	})
}

// GetTaxCodes obtiene códigos de impuestos disponibles
func (h *ComprobanteHandler) GetTaxCodes(c *gin.Context) {
	taxCodes := map[string]interface{}{
		"afectacion_igv": map[string]string{
			"10": "Gravado - Operación Onerosa",
			"11": "Gravado - Retiro por premio",
			"12": "Gravado - Retiro por donación",
			"13": "Gravado - Retiro",
			"14": "Gravado - Retiro por publicidad",
			"15": "Gravado - Bonificaciones",
			"16": "Gravado - Retiro por entrega a trabajadores",
			"17": "Gravado - IVAP",
			"20": "Exonerado - Operación Onerosa",
			"21": "Exonerado - Transferencia Gratuita",
			"30": "Inafecto - Operación Onerosa",
			"31": "Inafecto - Retiro por Bonificación",
			"32": "Inafecto - Retiro",
			"33": "Inafecto - Retiro por Muestras Médicas",
			"34": "Inafecto - Retiro por Convenio Colectivo",
			"35": "Inafecto - Retiro por premio",
			"36": "Inafecto - Retiro por publicidad",
			"37": "Inafecto - Transferencia Gratuita",
			"40": "Exportación",
		},
		"tipos_impuesto": map[string]string{
			"1000": "IGV",
			"2000": "ISC",
			"9999": "OTROS",
		},
		"tipos_documento": map[string]string{
			"01": "Factura",
			"03": "Boleta de Venta",
			"07": "Nota de Crédito",
			"08": "Nota de Débito",
		},
	}

	c.JSON(http.StatusOK, taxCodes)
}

// GetDocumentTypes obtiene tipos de documento disponibles
func (h *ComprobanteHandler) GetDocumentTypes(c *gin.Context) {
	documentTypes := map[string]interface{}{
		"comprobantes": map[string]string{
			"1": "Factura",
			"2": "Boleta",
			"3": "Nota de Crédito",
			"4": "Nota de Débito",
		},
		"identidad": map[string]string{
			"0": "DOC.TRIB.NO.DOM.SIN.RUC",
			"1": "DNI",
			"4": "CE",
			"6": "RUC",
			"7": "PASAPORTE",
			"A": "CED.DIPLOMATICA DE IDENTIDAD",
		},
		"monedas": map[string]string{
			"PEN": "Soles",
			"USD": "Dólares Americanos",
			"EUR": "Euros",
		},
		"unidades_medida": map[string]string{
			"NIU": "Unidad",
			"KGM": "Kilogramo",
			"GRM": "Gramo",
			"LTR": "Litro",
			"MTR": "Metro",
			"ZZ":  "Servicio",
		},
	}

	c.JSON(http.StatusOK, documentTypes)
}

// Métodos para manejo de lotes

// SendBatch envía múltiples comprobantes en lote
func (h *ComprobanteHandler) SendBatch(c *gin.Context) {
	var request struct {
		ComprobanteIDs      []string `json:"comprobante_ids" binding:"required"`
		CertificatePath     string   `json:"certificate_path" binding:"required"`
		CertificatePassword string   `json:"certificate_password" binding:"required"`
		Description         string   `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos inválidos",
			"details": err.Error(),
		})
		return
	}

	// Generar ID del lote
	batchID := uuid.New().String()

	// Crear lote en base de datos
	fechaInicio := time.Now()
	batch := &repository.Lote{
		ID:               batchID,
		Descripcion:      request.Description,
		TotalDocumentos:  len(request.ComprobanteIDs),
		Estado:           "PROCESANDO",
		FechaInicio:      &fechaInicio,
		FechaCreacion:    time.Now(),
	}

	if err := h.repository.CreateBatch(batch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error creando lote",
		})
		return
	}

	// Procesar lote en goroutine
	go h.processBatch(batchID, request.ComprobanteIDs, request.CertificatePath, request.CertificatePassword)

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Lote creado y procesándose",
		"batch_id": batchID,
		"total_documents": len(request.ComprobanteIDs),
	})
}

// GetBatchStatus obtiene el estado de un lote
func (h *ComprobanteHandler) GetBatchStatus(c *gin.Context) {
	batchID := c.Param("batch_id")

	batch, err := h.repository.GetBatch(batchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Lote no encontrado",
		})
		return
	}

	c.JSON(http.StatusOK, batch)
}

// processBatch procesa un lote de comprobantes en segundo plano
func (h *ComprobanteHandler) processBatch(batchID string, comprobanteIDs []string, certPath, certPass string) {
	var exitosos, fallidos int

	for _, id := range comprobanteIDs {
		// Procesar cada comprobante
		if err := h.processComprobanteForBatch(id, certPath, certPass); err != nil {
			fallidos++
		} else {
			exitosos++
		}

		// Actualizar progreso del lote
		h.repository.UpdateBatchProgress(batchID, exitosos+fallidos, exitosos, fallidos)
	}

	// Finalizar lote
	estado := "COMPLETADO"
	if fallidos > 0 {
		estado = "COMPLETADO_CON_ERRORES"
	}

	h.repository.FinalizeBatch(batchID, estado, time.Now())
}

// processComprobanteForBatch procesa un comprobante individual dentro de un lote
func (h *ComprobanteHandler) processComprobanteForBatch(id, certPath, certPass string) error {
	// Obtener comprobante
	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		return err
	}

	// Procesar si está pendiente
	if comprobante.EstadoProceso == models.EstadoPendiente {
		// Convertir a UBL
		ublDocument, err := h.conversionService.ConvertToUBL(comprobante)
		if err != nil {
			return err
		}

		// Serializar a XML
		xmlData, err := h.conversionService.UBLService.SerializeToXML(ublDocument)
		if err != nil {
			return err
		}

		// Guardar XML
		if err := h.repository.UpdateXML(id, string(xmlData)); err != nil {
			return err
		}

		comprobante.XMLGenerado = string(xmlData)
	}

	// Firmar si no está firmado
	if comprobante.EstadoProceso < models.EstadoFirmado {
		xmlData := []byte(comprobante.XMLGenerado)
		signedXML, err := h.signingService.SignDocument(xmlData, certPath, certPass)
		if err != nil {
			return err
		}

		if err := h.repository.UpdateSignedXML(id, string(signedXML)); err != nil {
			return err
		}

		comprobante.XMLFirmado = string(signedXML)
	}

	// Enviar a SUNAT
	if comprobante.EstadoProceso == models.EstadoFirmado {
		documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero

		pkg, err := h.encodingService.ProcessForSUNAT([]byte(comprobante.XMLFirmado), documentID)
		if err != nil {
			return err
		}

		response, err := h.sunatService.SendDocument(pkg)
		if err != nil {
			return err
		}

		// Actualizar información SUNAT
		if err := h.repository.UpdateSUNATInfo(id, response.Ticket, "ENVIADO"); err != nil {
			return err
		}

		if err := h.repository.UpdateStatus(id, models.EstadoEnviado); err != nil {
			return err
		}
	}

	return nil
}

// FullProcessComprobante realiza conversión a UBL, firmado y empaquetado ZIP en un solo endpoint
func (h *ComprobanteHandler) FullProcessComprobante(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	// 1. Generar estructura UBL
	ublStruct, err := h.conversionService.ConvertToUBL(comprobante)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error generando estructura UBL",
			"details": err.Error(),
		})
		return
	}
	// 2. Serializar a XML
	xmlUBL, err := h.conversionService.UBLService.SerializeToXML(ublStruct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error serializando UBL a XML",
			"details": err.Error(),
		})
		return
	}
	// 3. Firmar XML (usando certificado por defecto)
	certPath := config.AppConfig.Security.CertificatePath
	certPass := config.AppConfig.Security.CertificatePass
	xmlFirmado, err := h.signingService.SignDocument(xmlUBL, certPath, certPass)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error firmando XML",
			"details": err.Error(),
		})
		return
	}
	// 4. Empaquetar en ZIP (base64)
	documentID := comprobante.Emisor.RUC + "-" + comprobante.Tipo.String() + "-" + comprobante.Serie + "-" + comprobante.Numero
	zipPkg, err := h.encodingService.ProcessForSUNAT(xmlFirmado, documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error empaquetando ZIP",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"xml_ubl":     string(xmlUBL),
		"xml_firmado": string(xmlFirmado),
		"zip_base64":  zipPkg.Base64Content,
	})
}

// GetComprobanteResult devuelve el estado, XML firmado, ZIP y CDR del comprobante
func (h *ComprobanteHandler) GetComprobanteResult(c *gin.Context) {
	id := c.Param("id")

	comprobante, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Comprobante no encontrado",
		})
		return
	}

	zipBase64 := ""
	if comprobante.ArchivoZIP != nil && len(comprobante.ArchivoZIP) > 0 {
		zipBase64 = h.encodingService.EncodeToBase64(comprobante.ArchivoZIP)
	}
	cdrBase64 := ""
	if comprobante.CDRSUNAT != nil && len(comprobante.CDRSUNAT) > 0 {
		cdrBase64 = h.encodingService.EncodeToBase64(comprobante.CDRSUNAT)
	}

	c.JSON(http.StatusOK, gin.H{
		"estado":      comprobante.EstadoSUNAT,
		"xml_firmado": comprobante.XMLFirmado,
		"zip_base64":  zipBase64,
		"cdr_base64":  cdrBase64,
	})
}

// Consulta de contribuyente por RUC
func (h *ComprobanteHandler) GetContribuyente(c *gin.Context) {
	ruc := c.Query("ruc")
	if ruc == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro ruc requerido"})
		return
	}
	// Respuesta mock (puedes conectar a una BD real si lo deseas)
	c.JSON(http.StatusOK, gin.H{
		"ruc":          ruc,
		"razon_social": "EMPRESA DEMO S.A.C.",
		"estado":       "ACTIVO",
		"direccion":    "AV. AREQUIPA 123, LIMA",
	})
}


// Consulta de tasa de cambio (mock)
func (h *ComprobanteHandler) GetTasaCambio(c *gin.Context) {
	moneda := c.DefaultQuery("moneda", "USD")
	fecha := c.DefaultQuery("fecha", time.Now().Format("02-01-2006"))
	// Respuesta mock (puedes conectar a un servicio real si lo deseas)
	c.JSON(http.StatusOK, gin.H{
		"moneda": moneda,
		"fecha":  fecha,
		"tasa":   3.85,
	})
}

// Calculadora de cambio (mock)
func (h *ComprobanteHandler) CalculadoraCambio(c *gin.Context) {
	valor, _ := strconv.ParseFloat(c.DefaultQuery("valor", "0"), 64)
	de := c.DefaultQuery("de", "USD")
	a := c.DefaultQuery("a", "PEN")
	fecha := c.DefaultQuery("fecha", time.Now().Format("02-01-2006"))
	tasa := 3.85 // valor mock
	if de == "PEN" && a == "USD" {
		tasa = 1 / tasa
	}
	resultado := valor * tasa
	c.JSON(http.StatusOK, gin.H{
		"valor_inicial": valor,
		"de":            de,
		"a":             a,
		"fecha":         fecha,
		"tasa":          tasa,
		"resultado":     resultado,
	})
}

// Handler RESTful para crear factura
func (h *ComprobanteHandler) CreateFactura(c *gin.Context) {
	logrus.Info("[API] Creando factura electrónica")
	h.createComprobanteTipoValidado(c, 1) // 1 = Factura
}

// Handler RESTful para crear nota de crédito
func (h *ComprobanteHandler) CreateNotaCredito(c *gin.Context) {
	logrus.Info("[API] Creando nota de crédito electrónica")
	h.createComprobanteTipoValidado(c, 3) // 3 = Nota de Crédito
}

// Handler RESTful para crear nota de débito
func (h *ComprobanteHandler) CreateNotaDebito(c *gin.Context) {
	logrus.Info("[API] Creando nota de débito electrónica")
	h.createComprobanteTipoValidado(c, 4) // 4 = Nota de Débito
}

// Lógica común para crear comprobante RESTful con validación reforzada
func (h *ComprobanteHandler) createComprobanteTipoValidado(c *gin.Context, tipo int) {
	var comprobante models.Comprobante
	if err := c.ShouldBindJSON(&comprobante); err != nil {
		logrus.WithError(err).Warn("Datos inválidos en request JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos", "details": err.Error()})
		return
	}
	comprobante.Tipo = models.TipoComprobante(tipo)

	// Validar RUC emisor
	if comprobante.Emisor.RUC == "" || len(comprobante.Emisor.RUC) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "RUC emisor vacío o inválido (debe tener 11 dígitos)"})
		return
	}
	// Validar razón social emisor
	if strings.TrimSpace(comprobante.Emisor.RazonSocial) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Razón social del emisor es obligatoria"})
		return
	}
	// Validar dirección emisor
	if strings.TrimSpace(comprobante.Emisor.Direccion) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dirección del emisor es obligatoria"})
		return
	}
	// Validar RUC receptor
	if comprobante.Receptor.NumeroDocumento == "" || len(comprobante.Receptor.NumeroDocumento) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "RUC receptor vacío o inválido (debe tener 11 dígitos)"})
		return
	}
	// Validar razón social receptor
	if strings.TrimSpace(comprobante.Receptor.RazonSocial) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Razón social del receptor es obligatoria"})
		return
	}

	// Generar ID único y fechas
	comprobante.ID = uuid.New().String()
	comprobante.FechaCreacion = time.Now()
	comprobante.FechaActualizacion = time.Now()
	comprobante.EstadoProceso = models.EstadoPendiente
	// Calcular totales automáticamente
	if err := h.conversionService.CalculateTotals(&comprobante); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error calculando totales", "details": err.Error()})
		return
	}
	// Guardar en base de datos (cabecera y detalle)
	if err := h.repository.Create(&comprobante); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error guardando comprobante", "details": err.Error()})
		return
	}
	// 1. Generar estructura UBL
	ublStruct, err := h.conversionService.ConvertToUBL(&comprobante)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error generando UBL", "details": err.Error()})
		return
	}
	// 2. Serializar a XML
	xmlUBL, err := h.conversionService.UBLService.SerializeToXML(ublStruct)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error serializando UBL a XML", "details": err.Error()})
		return
	}
	// 3. Firmar XML usando el certificado configurado
	certPath := config.AppConfig.Security.CertificatePath
	certPass := config.AppConfig.Security.CertificatePass
	xmlFirmado, err := h.signingService.SignDocument(xmlUBL, certPath, certPass)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error firmando XML", "details": err.Error()})
		return
	}
	// Guardar el XML firmado en la base de datos
	if err := h.repository.UpdateSignedXML(comprobante.ID, string(xmlFirmado)); err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error guardando XML firmado", "details": err.Error()})
		return
	}
	// 4. Empaquetar en ZIP (base64)
	documentID := comprobante.Emisor.RUC + "-" + strconv.Itoa(int(comprobante.Tipo)) + "-" + comprobante.Serie + "-" + comprobante.Numero
	zipPkg, err := h.encodingService.ProcessForSUNAT(xmlFirmado, documentID)
	if err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error empaquetando ZIP", "details": err.Error()})
		return
	}
	if err := h.repository.UpdateArchivoZIP(comprobante.ID, zipPkg.ZipContent); err != nil {
		h.repository.UpdateStatus(comprobante.ID, models.EstadoError)
		c.JSON(500, gin.H{"error": "Error guardando ZIP", "details": err.Error()})
		return
	}
	// Guardar archivos en xml_pruebas
	fileNameXML := documentID + "-signed.xml"
	fileNameZIP := documentID + ".zip"
	_ = saveToXMLPruebas(fileNameXML, xmlFirmado)
	_ = saveToXMLPruebas(fileNameZIP, zipPkg.ZipContent)
	// Responder con el comprobante, XML firmado y ZIP (en base64)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Comprobante creado, firmado y zipeado exitosamente",
		"id":          comprobante.ID,
		"xml_firmado": string(xmlFirmado),
		"zip_base64":  zipPkg.Base64Content,
		"file_xml":    fileNameXML,
		"file_zip":    fileNameZIP,
	})
}

// Endpoint para descargar PDF (mock)
func (h *ComprobanteHandler) DownloadPDF(c *gin.Context) {
	id := c.Param("id")
	// Simulación de URL de PDF
	c.JSON(200, gin.H{
		"pdf_url": "https://tuservidor.com/pdfs/" + id + ".pdf",
	})
}