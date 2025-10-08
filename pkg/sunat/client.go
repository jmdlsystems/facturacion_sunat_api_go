package sunat

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"facturacion_sunat_api_go/internal/config"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	config     *config.SUNATConfig
	httpClient *http.Client
	baseURL    string
}

type SOAPEnvelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	Xmlns   string   `xml:"xmlns:soap,attr"`
	Header  SOAPHeader `xml:"soap:Header"`
	Body    SOAPBody `xml:"soap:Body"`
}

type SOAPHeader struct {
	Security Security `xml:"wsse:Security"`
}

type Security struct {
	XMLName       xml.Name      `xml:"wsse:Security"`
	Xmlns         string        `xml:"xmlns:wsse,attr"`
	UsernameToken UsernameToken `xml:"wsse:UsernameToken"`
}

type UsernameToken struct {
	Username string `xml:"wsse:Username"`
	Password string `xml:"wsse:Password"`
}

type SOAPBody struct {
	SendBill   *SendBillRequest   `xml:"sendBill,omitempty"`
	GetStatus  *GetStatusRequest  `xml:"getStatus,omitempty"`
	GetCDR     *GetCDRRequest     `xml:"getStatusCdr,omitempty"`
}

type SendBillRequest struct {
	XMLName     xml.Name `xml:"sendBill"`
	Xmlns       string   `xml:"xmlns,attr"`
	FileName    string   `xml:"fileName"`
	ContentFile string   `xml:"contentFile"`
}

type GetStatusRequest struct {
	XMLName xml.Name `xml:"getStatus"`
	Xmlns   string   `xml:"xmlns,attr"`
	Ticket  string   `xml:"ticket"`
}

type GetCDRRequest struct {
	XMLName xml.Name `xml:"getStatusCdr"`
	Xmlns   string   `xml:"xmlns,attr"`
	RUC     string   `xml:"rucComprobante"`
	Tipo    string   `xml:"tipoComprobante"`
	Serie   string   `xml:"serieComprobante"`
	Numero  string   `xml:"numeroComprobante"`
}

// Response structures
type SOAPResponse struct {
	XMLName xml.Name         `xml:"Envelope"`
	Body    SOAPResponseBody `xml:"Body"`
}

type SOAPResponseBody struct {
	SendBillResponse   *SendBillResponse   `xml:"sendBillResponse,omitempty"`
	GetStatusResponse  *GetStatusResponse  `xml:"getStatusResponse,omitempty"`
	GetCDRResponse     *GetCDRResponse     `xml:"getStatusCdrResponse,omitempty"`
	Fault              *SOAPFault          `xml:"Fault,omitempty"`
}

type SendBillResponse struct {
	ApplicationResponse string `xml:"applicationResponse"`
	Ticket              string `xml:"ticket"`
}

type GetStatusResponse struct {
	Status      string `xml:"status"`
	StatusCode  string `xml:"statusCode"`
	Content     string `xml:"content"`
}

type GetCDRResponse struct {
	Content     string `xml:"content"`
	StatusCode  string `xml:"statusCode"`
}

type SOAPFault struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
	Detail      string `xml:"detail"`
}

func NewClient(cfg *config.SUNATConfig) *Client {
	// Configurar cliente HTTP con timeouts y SSL
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	// Determinar URL base
	baseURL := cfg.BaseURL
	if cfg.BetaURL != "" {
		baseURL = cfg.BetaURL // Usar beta por defecto en desarrollo
	}

	return &Client{
		config:     cfg,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// SendDocument envía un documento a SUNAT
func (c *Client) SendDocument(request *SendBillRequest) (*http.Response, error) {
	envelope := c.createSOAPEnvelope()
	envelope.Body.SendBill = request
	envelope.Body.SendBill.Xmlns = "http://service.sunat.gob.pe"

	return c.sendSOAPRequest(envelope, "sendBill")
}

// GetStatus consulta el estado de un documento
func (c *Client) GetStatus(request *GetStatusRequest) (*http.Response, error) {
	envelope := c.createSOAPEnvelope()
	envelope.Body.GetStatus = request
	envelope.Body.GetStatus.Xmlns = "http://service.sunat.gob.pe"

	return c.sendSOAPRequest(envelope, "getStatus")
}

// DownloadCDR descarga el CDR de un documento
func (c *Client) DownloadCDR(request *GetCDRRequest) (*http.Response, error) {
	envelope := c.createSOAPEnvelope()
	envelope.Body.GetCDR = request
	envelope.Body.GetCDR.Xmlns = "http://service.sunat.gob.pe"

	return c.sendSOAPRequest(envelope, "getStatusCdr")
}

// ValidateRUC valida un RUC (funcionalidad extendida)
func (c *Client) ValidateRUC(ruc string) (*http.Response, error) {
	// TODO: Implementar integración real con endpoint de SUNAT para validación de RUC
	return nil, fmt.Errorf("validación de RUC no implementada aún")
}

// TestConnection prueba la conectividad con SUNAT
func (c *Client) TestConnection(request *http.Request) (*http.Response, error) {
	// Crear un request simple para probar conectividad
	testReq, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request de prueba: %v", err)
	}

	resp, err := c.httpClient.Do(testReq)
	if err != nil {
		return nil, fmt.Errorf("error en conexión de prueba: %v", err)
	}

	return resp, nil
}

// GetServiceStatus obtiene el estado del servicio SUNAT
func (c *Client) GetServiceStatus() (*http.Response, error) {
	return c.TestConnection(nil)
}

// createSOAPEnvelope crea el envelope SOAP básico con autenticación
func (c *Client) createSOAPEnvelope() *SOAPEnvelope {
	return &SOAPEnvelope{
		Xmlns: "http://schemas.xmlsoap.org/soap/envelope/",
		Header: SOAPHeader{
			Security: Security{
				Xmlns: "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd",
				UsernameToken: UsernameToken{
					Username: c.config.Username,
					Password: c.config.Password,
				},
			},
		},
		Body: SOAPBody{},
	}
}

// sendSOAPRequest envía una request SOAP a SUNAT
func (c *Client) sendSOAPRequest(envelope *SOAPEnvelope, action string) (*http.Response, error) {
	// Serializar envelope a XML
	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializando SOAP envelope: %v", err)
	}

	// Agregar declaración XML
	xmlWithHeader := append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`), xmlData...)

	fmt.Printf("🔍 Enviando request SOAP a: %s\n", c.baseURL)
	fmt.Printf("🔍 Action: %s\n", action)
	fmt.Printf("🔍 Request XML:\n%s\n", string(xmlWithHeader))

	// Crear HTTP request
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(xmlWithHeader))
	if err != nil {
		return nil, fmt.Errorf("error creando HTTP request: %v", err)
	}

	// Configurar headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", fmt.Sprintf("urn:%s", action))
	req.Header.Set("User-Agent", "FacturacionElectronica/1.0")

	// Realizar request con reintentos
	var resp *http.Response
	for i := 0; i <= c.config.MaxRetries; i++ {
		resp, err = c.httpClient.Do(req)
		if err == nil {
			break
		}

		if i < c.config.MaxRetries {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error enviando request a SUNAT después de %d intentos: %v", c.config.MaxRetries+1, err)
	}

	fmt.Printf("🔍 Respuesta recibida - Status: %d\n", resp.StatusCode)
	return resp, nil
}

// ParseSOAPResponse parsea una respuesta SOAP de SUNAT
func (c *Client) ParseSOAPResponse(resp *http.Response) (*SOAPResponse, error) {
	// NO cerrar el body aquí, ya que el servicio lo necesita leer
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %v", err)
	}

	var soapResp SOAPResponse
	if err := xml.Unmarshal(body, &soapResp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta SOAP: %v", err)
	}

	// Verificar si hay errores SOAP
	if soapResp.Body.Fault != nil {
		return nil, fmt.Errorf("SOAP Fault: %s - %s", 
			soapResp.Body.Fault.FaultCode, 
			soapResp.Body.Fault.FaultString)
	}

	return &soapResp, nil
}

// CreateSendBillRequest crea un request para envío de comprobantes
func (c *Client) CreateSendBillRequest(fileName, content string) *SendBillRequest {
	return &SendBillRequest{
		FileName:    fileName,
		ContentFile: content,
	}
}

// CreateGetStatusRequest crea un request para consulta de estado
func (c *Client) CreateGetStatusRequest(ticket string) *GetStatusRequest {
	return &GetStatusRequest{
		Ticket: ticket,
	}
}

// CreateGetCDRRequest crea un request para descarga de CDR
func (c *Client) CreateGetCDRRequest(ruc, tipo, serie, numero string) *GetCDRRequest {
	return &GetCDRRequest{
		RUC:    ruc,
		Tipo:   tipo,
		Serie:  serie,
		Numero: numero,
	}
}

// IsProduction verifica si estamos en ambiente de producción
func (c *Client) IsProduction() bool {
	return c.baseURL == c.config.BaseURL && c.config.BaseURL != c.config.BetaURL
}

// GetEnvironment retorna el ambiente actual
func (c *Client) GetEnvironment() string {
	if c.IsProduction() {
		return "production"
	}
	return "beta"
}

// GetBaseURL retorna la URL base del cliente
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// Ping verifica la conectividad con SUNAT
func (c *Client) Ping() error {
	resp, err := c.TestConnection(nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SUNAT respondió con status: %d", resp.StatusCode)
	}
	
	return nil
}