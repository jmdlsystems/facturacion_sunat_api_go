package handlers

import (
	"bytes"
	"encoding/json"
	"facturacion_sunat_api_go/internal/models"
	"facturacion_sunat_api_go/internal/repository"
	"facturacion_sunat_api_go/internal/services"
	"facturacion_sunat_api_go/pkg/certificate"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository es un mock del repositorio para pruebas
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(comprobante *models.Comprobante) error {
	args := m.Called(comprobante)
	return args.Error(0)
}

func (m *MockRepository) GetByID(id string) (*models.Comprobante, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Comprobante), args.Error(1)
}

func (m *MockRepository) List() ([]models.Comprobante, error) {
	args := m.Called()
	return args.Get(0).([]models.Comprobante), args.Error(1)
}

func (m *MockRepository) UpdateStatus(id string, status models.EstadoProceso) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateXML(id, xml string) error {
	args := m.Called(id, xml)
	return args.Error(0)
}

func (m *MockRepository) UpdateSignedXML(id, signedXML string) error {
	args := m.Called(id, signedXML)
	return args.Error(0)
}

func (m *MockRepository) UpdateArchivoZIP(id string, zipContent []byte) error {
	args := m.Called(id, zipContent)
	return args.Error(0)
}

func (m *MockRepository) UpdateSUNATInfo(id, ticket, status string) error {
	args := m.Called(id, ticket, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateCDR(id string, cdr []byte) error {
	args := m.Called(id, cdr)
	return args.Error(0)
}

func (m *MockRepository) UpdateError(id, error string) error {
	args := m.Called(id, error)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestGenerateXMLWithoutSignature prueba la generación de XML sin firma
func TestGenerateXMLWithoutSignature(t *testing.T) {
	// Configurar Gin para pruebas
	gin.SetMode(gin.TestMode)

	// Crear mock del repositorio
	mockRepo := new(MockRepository)

	// Crear servicios reales
	certManager := certificate.NewManager()
	ublService := services.NewUBLService()
	conversionService := services.NewConversionService(ublService)
	signingService := services.NewSigningService(certManager, ublService)
	encodingService := services.NewEncodingService()
	sunatService := services.NewSUNATService(&models.SUNATConfig{}, encodingService)

	// Crear handler
	handler := NewComprobanteHandler(
		mockRepo,
		conversionService,
		signingService,
		encodingService,
		sunatService,
	)

	// Crear comprobante de prueba válido
	comprobante := models.Comprobante{
		Tipo:        models.TipoFactura,
		Serie:       "F001",
		Numero:      "00000001",
		TipoMoneda:  "PEN",
		FechaEmision: time.Now(),
		Emisor: models.Emisor{
			RUC:     "20123456789",
			RazonSocial: "EMPRESA DE PRUEBA SAC",
			NombreComercial: "EMPRESA DE PRUEBA",
			Direccion: "AV. AREQUIPA 123",
			Distrito: "LIMA",
			Provincia: "LIMA",
			Departamento: "LIMA",
			Pais: "PE",
		},
		Receptor: models.Receptor{
			TipoDocumento:    "1", // DNI
			NumeroDocumento:  "12345678",
			RazonSocial:      "CLIENTE DE PRUEBA",
			Direccion:        "AV. TEST 456",
			Distrito:         "LIMA",
			Provincia:        "LIMA",
			Departamento:     "LIMA",
			Pais:             "PE",
		},
		Items: []models.Item{
			{
				Codigo:        "PROD001",
				Descripcion:   "PRODUCTO DE PRUEBA",
				Cantidad:      1,
				PrecioUnitario: 100.00,
				Impuestos: []models.ImpuestoItem{
					{
						Tipo:   "1000", // IGV
						Porcentaje: 18.0,
					},
				},
			},
		},
	}

	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	assert.NoError(t, err)

	// Crear request HTTP
	req, err := http.NewRequest("POST", "/api/v1/utils/generate-xml", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Crear response recorder
	w := httptest.NewRecorder()

	// Configurar router de prueba
	router := gin.New()
	router.POST("/api/v1/utils/generate-xml", handler.GenerateXMLWithoutSignature)

	// Ejecutar request
	router.ServeHTTP(w, req)

	// Verificar respuesta
	assert.Equal(t, http.StatusOK, w.Code)

	// Parsear respuesta JSON
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verificar campos de respuesta
	assert.Equal(t, "XML generado sin firma exitosamente", response["message"])
	assert.Contains(t, response, "document_id")
	assert.Contains(t, response, "file_name")
	assert.Contains(t, response, "xml_content")
	assert.Contains(t, response, "totals")

	// Verificar que el XML contiene elementos UBL básicos
	xmlContent := response["xml_content"].(string)
	assert.Contains(t, xmlContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, xmlContent, "<Invoice")
	assert.Contains(t, xmlContent, "xmlns=\"urn:oasis:names:specification:ubl:schema:xsd:Invoice-2\"")
	assert.Contains(t, xmlContent, "<cbc:UBLVersionID>2.1</cbc:UBLVersionID>")
	assert.Contains(t, xmlContent, "<cbc:ID>F001-00000001</cbc:ID>")
	assert.Contains(t, xmlContent, "<cbc:DocumentCurrencyCode>PEN</cbc:DocumentCurrencyCode>")

	// Verificar que NO contiene elementos de firma
	assert.NotContains(t, xmlContent, "<ds:Signature")
	assert.NotContains(t, xmlContent, "<ext:UBLExtensions>")
}

// TestGenerateXMLWithoutSignatureInvalidData prueba con datos inválidos
func TestGenerateXMLWithoutSignatureInvalidData(t *testing.T) {
	// Configurar Gin para pruebas
	gin.SetMode(gin.TestMode)

	// Crear mock del repositorio
	mockRepo := new(MockRepository)

	// Crear servicios reales
	certManager := certificate.NewManager()
	ublService := services.NewUBLService()
	conversionService := services.NewConversionService(ublService)
	signingService := services.NewSigningService(certManager, ublService)
	encodingService := services.NewEncodingService()
	sunatService := services.NewSUNATService(&models.SUNATConfig{}, encodingService)

	// Crear handler
	handler := NewComprobanteHandler(
		mockRepo,
		conversionService,
		signingService,
		encodingService,
		sunatService,
	)

	// Crear comprobante inválido (sin RUC)
	comprobante := models.Comprobante{
		Tipo:        models.TipoFactura,
		Serie:       "F001",
		Numero:      "00000001",
		TipoMoneda:  "PEN",
		FechaEmision: time.Now(),
		Emisor: models.Emisor{
			RUC:     "", // RUC vacío - inválido
			RazonSocial: "EMPRESA DE PRUEBA SAC",
		},
		Receptor: models.Receptor{
			TipoDocumento:    "1",
			NumeroDocumento:  "12345678",
			RazonSocial:      "CLIENTE DE PRUEBA",
		},
		Items: []models.Item{
			{
				Codigo:        "PROD001",
				Descripcion:   "PRODUCTO DE PRUEBA",
				Cantidad:      1,
				PrecioUnitario: 100.00,
			},
		},
	}

	// Convertir a JSON
	jsonData, err := json.Marshal(comprobante)
	assert.NoError(t, err)

	// Crear request HTTP
	req, err := http.NewRequest("POST", "/api/v1/comprobantes/generate-xml", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Crear response recorder
	w := httptest.NewRecorder()

	// Configurar router de prueba
	router := gin.New()
	router.POST("/api/v1/comprobantes/generate-xml", handler.GenerateXMLWithoutSignature)

	// Ejecutar request
	router.ServeHTTP(w, req)

	// Verificar que devuelve error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Parsear respuesta JSON
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verificar que contiene mensaje de error
	assert.Contains(t, response, "error")
	assert.Contains(t, response["error"], "RUC emisor es obligatorio")
}

// TestGenerateXMLWithoutSignatureDifferentTypes prueba diferentes tipos de comprobantes
func TestGenerateXMLWithoutSignatureDifferentTypes(t *testing.T) {
	// Configurar Gin para pruebas
	gin.SetMode(gin.TestMode)

	// Crear mock del repositorio
	mockRepo := new(MockRepository)

	// Crear servicios reales
	certManager := certificate.NewManager()
	ublService := services.NewUBLService()
	conversionService := services.NewConversionService(ublService)
	signingService := services.NewSigningService(certManager, ublService)
	encodingService := services.NewEncodingService()
	sunatService := services.NewSUNATService(&models.SUNATConfig{}, encodingService)

	// Crear handler
	handler := NewComprobanteHandler(
		mockRepo,
		conversionService,
		signingService,
		encodingService,
		sunatService,
	)

	// Casos de prueba para diferentes tipos
	testCases := []struct {
		name      string
		tipo      models.TipoComprobante
		expectedElement string
	}{
		{"Factura", models.TipoFactura, "<Invoice"},
		{"Boleta", models.TipoBoleta, "<Invoice"},
		{"Nota de Crédito", models.TipoNotaCredito, "<CreditNote"},
		{"Nota de Débito", models.TipoNotaDebito, "<DebitNote"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Crear comprobante con el tipo específico
			comprobante := models.Comprobante{
				Tipo:        tc.tipo,
				Serie:       "F001",
				Numero:      "00000001",
				TipoMoneda:  "PEN",
				FechaEmision: time.Now(),
				Emisor: models.Emisor{
					RUC:     "20123456789",
					RazonSocial: "EMPRESA DE PRUEBA SAC",
				},
				Receptor: models.Receptor{
					TipoDocumento:    "1",
					NumeroDocumento:  "12345678",
					RazonSocial:      "CLIENTE DE PRUEBA",
				},
				Items: []models.Item{
					{
						Codigo:        "PROD001",
						Descripcion:   "PRODUCTO DE PRUEBA",
						Cantidad:      1,
						PrecioUnitario: 100.00,
						Impuestos: []models.ImpuestoItem{
							{
								Tipo:   "1000",
								Porcentaje: 18.0,
							},
						},
					},
				},
			}

			// Convertir a JSON
			jsonData, err := json.Marshal(comprobante)
			assert.NoError(t, err)

			// Crear request HTTP
			req, err := http.NewRequest("POST", "/api/v1/utils/generate-xml", bytes.NewBuffer(jsonData))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Crear response recorder
			w := httptest.NewRecorder()

			// Configurar router de prueba
			router := gin.New()
			router.POST("/api/v1/utils/generate-xml", handler.GenerateXMLWithoutSignature)

			// Ejecutar request
			router.ServeHTTP(w, req)

			// Verificar respuesta exitosa
			assert.Equal(t, http.StatusOK, w.Code)

			// Parsear respuesta JSON
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verificar que el XML contiene el elemento esperado
			xmlContent := response["xml_content"].(string)
			assert.Contains(t, xmlContent, tc.expectedElement)
		})
	}
} 