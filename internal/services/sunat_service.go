package services

import (
	"encoding/xml"
	"facturacion_sunat_api_go/internal/config"
	"facturacion_sunat_api_go/pkg/sunat"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SUNATService struct {
	Client         *sunat.Client
	config         *config.SUNATConfig
	encodingService *EncodingService
}

func NewSUNATService(cfg *config.SUNATConfig, encodingService *EncodingService) *SUNATService {
	client := sunat.NewClient(cfg)
	
	return &SUNATService{
		Client:          client,
		config:          cfg,
		encodingService: encodingService,
	}
}

// SendDocument envía un documento a SUNAT
func (s *SUNATService) SendDocument(pkg *SUNATPackage) (*SUNATSendResponse, error) {
	// Validar paquete antes del envío
	if err := pkg.ValidatePackage(); err != nil {
		return nil, fmt.Errorf("paquete inválido: %v", err)
	}

	// Modo simulación para desarrollo/pruebas
	if s.isSimulationMode() {
		return s.simulateSUNATResponse(pkg)
	}

	// Preparar request usando el cliente SUNAT
	sendBillRequest := s.Client.CreateSendBillRequest(pkg.FileName, pkg.Base64Content)

	// Enviar a SUNAT
	response, err := s.Client.SendDocument(sendBillRequest)
	if err != nil {
		return nil, fmt.Errorf("error enviando a SUNAT: %v", err)
	}

	// Procesar respuesta
	return s.processResponse(response)
}

// SendBatch envía múltiples documentos en lote
func (s *SUNATService) SendBatch(packages []*SUNATPackage) ([]*SUNATSendResponse, error) {
	var responses []*SUNATSendResponse
	var errors []error

	for i, pkg := range packages {
		response, err := s.SendDocument(pkg)
		if err != nil {
			errors = append(errors, fmt.Errorf("error en documento %d: %v", i+1, err))
			continue
		}
		responses = append(responses, response)

		// Pausa entre envíos para evitar límites de tasa
		if i < len(packages)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if len(errors) > 0 {
		return responses, fmt.Errorf("errores en el lote: %v", errors)
	}

	return responses, nil
}

// GetDocumentStatus consulta el estado de un documento en SUNAT
func (s *SUNATService) GetDocumentStatus(ticket string) (*SUNATStatusResponse, error) {
	if ticket == "" {
		return nil, fmt.Errorf("ticket requerido")
	}

	statusRequest := s.Client.CreateGetStatusRequest(ticket)
	
	response, err := s.Client.GetStatus(statusRequest)
	if err != nil {
		return nil, fmt.Errorf("error consultando estado: %v", err)
	}

	return s.processStatusResponse(response)
}

// DownloadCDR descarga el CDR (Constancia de Recepción) de SUNAT
func (s *SUNATService) DownloadCDR(ruc, tipo, serie, numero string) (*CDRResponse, error) {
	cdrRequest := s.Client.CreateGetCDRRequest(ruc, tipo, serie, numero)
	
	response, err := s.Client.DownloadCDR(cdrRequest)
	if err != nil {
		return nil, fmt.Errorf("error descargando CDR: %v", err)
	}

	return s.processCDRResponse(response)
}

// ValidateRUC valida un RUC contra SUNAT
func (s *SUNATService) ValidateRUC(ruc string) (*RUCValidationResponse, error) {
	if len(ruc) != 11 {
		return nil, fmt.Errorf("RUC debe tener 11 dígitos")
	}
	response, err := s.Client.ValidateRUC(ruc)
	if err != nil {
		if err.Error() == "validación de RUC no implementada aún" {
			return nil, fmt.Errorf("validación de RUC contra SUNAT no implementada")
		}
		return nil, fmt.Errorf("error validando RUC: %v", err)
	}
	return s.processRUCValidationResponse(response)
}

// processResponse procesa la respuesta de envío
func (s *SUNATService) processResponse(response *http.Response) (*SUNATSendResponse, error) {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %v", err)
	}

	fmt.Printf("🔍 Respuesta SUNAT - Status: %d, Body: %s\n", response.StatusCode, string(body))

	if response.StatusCode != http.StatusOK {
		return &SUNATSendResponse{
			Success:    false,
			StatusCode: response.StatusCode,
			Message:    string(body),
			Timestamp:  time.Now(),
		}, nil
	}

	// Parsear respuesta SOAP usando el cliente SUNAT
	soapResponse, err := s.Client.ParseSOAPResponse(response)
	if err != nil {
		return nil, fmt.Errorf("error parseando respuesta SOAP: %v", err)
	}

	fmt.Printf("🔍 Respuesta SOAP parseada - SendBillResponse: %+v\n", soapResponse.Body.SendBillResponse)

	// Procesar CDR si está presente
	var cdrData []byte
	if soapResponse.Body.SendBillResponse != nil && soapResponse.Body.SendBillResponse.ApplicationResponse != "" {
		cdrData, err = s.encodingService.DecodeFromBase64(soapResponse.Body.SendBillResponse.ApplicationResponse)
		if err != nil {
			return nil, fmt.Errorf("error decodificando CDR: %v", err)
		}
	}

	return &SUNATSendResponse{
		Success:             true,
		StatusCode:          response.StatusCode,
		Ticket:              soapResponse.Body.SendBillResponse.Ticket,
		ApplicationResponse: cdrData,
		Message:             "Documento enviado exitosamente",
		Timestamp:           time.Now(),
	}, nil
}

// processStatusResponse procesa respuesta de consulta de estado
func (s *SUNATService) processStatusResponse(response *http.Response) (*SUNATStatusResponse, error) {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta de estado: %v", err)
	}

	var statusResponse StatusSOAPResponse
	if err := xml.Unmarshal(body, &statusResponse); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de estado: %v", err)
	}

	return &SUNATStatusResponse{
		StatusCode:  statusResponse.Body.GetStatusResponse.StatusCode,
		Status:      statusResponse.Body.GetStatusResponse.Status,
		Description: statusResponse.Body.GetStatusResponse.Description,
		Content:     statusResponse.Body.GetStatusResponse.Content,
		Timestamp:   time.Now(),
	}, nil
}

// processCDRResponse procesa respuesta de descarga de CDR
func (s *SUNATService) processCDRResponse(response *http.Response) (*CDRResponse, error) {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta de CDR: %v", err)
	}

	var cdrResponse CDRSOAPResponse
	if err := xml.Unmarshal(body, &cdrResponse); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de CDR: %v", err)
	}

	// Decodificar contenido del CDR
	cdrContent, err := s.encodingService.DecodeFromBase64(cdrResponse.Body.DownloadResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("error decodificando CDR: %v", err)
	}

	return &CDRResponse{
		Success:   true,
		Content:   cdrContent,
		Message:   cdrResponse.Body.DownloadResponse.Message,
		Timestamp: time.Now(),
	}, nil
}

// processRUCValidationResponse procesa respuesta de validación de RUC
func (s *SUNATService) processRUCValidationResponse(response *http.Response) (*RUCValidationResponse, error) {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta de validación RUC: %v", err)
	}

	var validationResponse RUCValidationSOAPResponse
	if err := xml.Unmarshal(body, &validationResponse); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de validación RUC: %v", err)
	}

	return &RUCValidationResponse{
		Valid:       validationResponse.Body.ValidateResponse.Valid,
		RazonSocial: validationResponse.Body.ValidateResponse.RazonSocial,
		Estado:      validationResponse.Body.ValidateResponse.Estado,
		Direccion:   validationResponse.Body.ValidateResponse.Direccion,
		Timestamp:   time.Now(),
	}, nil
}

// TestConnection prueba la conectividad con SUNAT
func (s *SUNATService) TestConnection() error {
	response, err := s.Client.TestConnection(nil)
	if err != nil {
		return fmt.Errorf("error en conexión de prueba: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("servicio SUNAT no disponible (status: %d)", response.StatusCode)
	}

	return nil
}

// GetServiceStatus obtiene el estado del servicio SUNAT
func (s *SUNATService) GetServiceStatus() (*ServiceStatusResponse, error) {
	response, err := s.Client.GetServiceStatus()
	if err != nil {
		return &ServiceStatusResponse{
			Available: false,
			Message:   fmt.Sprintf("Servicio no disponible: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	defer response.Body.Close()

	return &ServiceStatusResponse{
		Available: response.StatusCode == http.StatusOK,
		Message:   fmt.Sprintf("Servicio disponible (status: %d)", response.StatusCode),
		Timestamp: time.Now(),
	}, nil
}

// isSimulationMode verifica si estamos en modo simulación
func (s *SUNATService) isSimulationMode() bool {
	// Si force_real_send está habilitado, siempre enviar real
	if s.config.ForceRealSend {
		return false
	}
	
	// Detectar modo simulación por credenciales de prueba
	return (s.config.Username == "MODDATOS" || s.config.Username == "20103129061MODDATOS") && s.config.Password == "MODDATOS"
}

// simulateSUNATResponse simula una respuesta exitosa de SUNAT
func (s *SUNATService) simulateSUNATResponse(pkg *SUNATPackage) (*SUNATSendResponse, error) {
	fmt.Println("🎭 Modo simulación activado - Simulando respuesta exitosa de SUNAT")
	
	return &SUNATSendResponse{
		Success:             true,
		StatusCode:          200,
		Message:             "Documento aceptado por SUNAT (SIMULACIÓN)",
		Ticket:              "123456789",
		ApplicationResponse: []byte("ACEPTADO"),
		Timestamp:           time.Now(),
	}, nil
}

// Estructuras de respuesta SOAP (mantener compatibilidad)
type StatusSOAPResponse struct {
	Body StatusResponseBody `xml:"Body"`
}

type StatusResponseBody struct {
	GetStatusResponse StatusResponse `xml:"getStatusResponse"`
}

type StatusResponse struct {
	StatusCode  string `xml:"statusCode"`
	Status      string `xml:"status"`
	Description string `xml:"description"`
	Content     string `xml:"content"`
}

type CDRSOAPResponse struct {
	Body CDRResponseBody `xml:"Body"`
}

type CDRResponseBody struct {
	DownloadResponse DownloadResponse `xml:"downloadResponse"`
}

type DownloadResponse struct {
	Content string `xml:"content"`
	Message string `xml:"message"`
}

type RUCValidationSOAPResponse struct {
	Body RUCValidationResponseBody `xml:"Body"`
}

type RUCValidationResponseBody struct {
	ValidateResponse RUCValidateResponse `xml:"validateResponse"`
}

type RUCValidateResponse struct {
	Valid       bool   `xml:"valid"`
	RazonSocial string `xml:"razonSocial"`
	Estado      string `xml:"estado"`
	Direccion   string `xml:"direccion"`
}

// Estructuras de respuesta del servicio
type SUNATSendResponse struct {
	Success             bool      `json:"success"`
	StatusCode          int       `json:"status_code"`
	Ticket              string    `json:"ticket,omitempty"`
	ApplicationResponse []byte    `json:"-"`
	Message             string    `json:"message"`
	Timestamp           time.Time `json:"timestamp"`
}

type SUNATStatusResponse struct {
	StatusCode  string    `json:"status_code"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
}

type CDRResponse struct {
	Success   bool      `json:"success"`
	Content   []byte    `json:"-"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type RUCValidationResponse struct {
	Valid       bool      `json:"valid"`
	RazonSocial string    `json:"razon_social"`
	Estado      string    `json:"estado"`
	Direccion   string    `json:"direccion"`
	Timestamp   time.Time `json:"timestamp"`
}

type ServiceStatusResponse struct {
	Available bool      `json:"available"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}