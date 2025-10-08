package services

import (
	"encoding/base64"
	"encoding/xml"
	"facturacion_sunat_api_go/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSUNATServiceSendDocumentAccepted prueba el envío exitoso a SUNAT
func TestSUNATServiceSendDocumentAccepted(t *testing.T) {
	// Crear servidor mock que simula SUNAT
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verificar que es una petición POST
		assert.Equal(t, "POST", r.Method)
		
		// Verificar Content-Type
		assert.Contains(t, r.Header.Get("Content-Type"), "text/xml")
		
		// Leer el body de la petición
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		bodyStr := string(body)
		
		// Verificar que contiene elementos SOAP esperados
		assert.Contains(t, bodyStr, "<soap:Envelope")
		assert.Contains(t, bodyStr, "<sendBill")
		assert.Contains(t, bodyStr, "<fileName>")
		assert.Contains(t, bodyStr, "<contentFile>")
		
		// Simular respuesta exitosa de SUNAT
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		// Respuesta SOAP de aceptación (ejemplo real de SUNAT)
		response := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ns2:sendBillResponse xmlns:ns2="http://service.sunat.gob.pe">
      <applicationResponse>` + base64.StdEncoding.EncodeToString([]byte("CDR-ACEPTADO")) + `</applicationResponse>
    </ns2:sendBillResponse>
  </soap:Body>
</soap:Envelope>`
		
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Crear configuración SUNAT con el servidor mock
	sunatConfig := &models.SUNATConfig{
		Endpoint: server.URL,
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Crear paquete de prueba
	testPackage := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Enviar documento
	response, err := sunatService.SendDocument(testPackage)

	// Verificar que no hay error
	assert.NoError(t, err)
	assert.NotNil(t, response)
	
	// Verificar que la respuesta indica aceptación
	assert.Contains(t, string(response.CDR), "CDR-ACEPTADO")
	assert.Equal(t, "ACEPTADO", response.Status)
}

// TestSUNATServiceSendDocumentRejected prueba el envío rechazado por SUNAT
func TestSUNATServiceSendDocumentRejected(t *testing.T) {
	// Crear servidor mock que simula SUNAT rechazando
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simular respuesta de rechazo de SUNAT
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		// Respuesta SOAP de rechazo
		rejectionCDR := `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2">
  <cbc:ReferenceID>20123456789-01-F001-00000001</cbc:ReferenceID>
  <cbc:ResponseCode>0003</cbc:ResponseCode>
  <cbc:Description>El Comprobante ingresado no cumple con el estandar establecido.</cbc:Description>
</ApplicationResponse>`
		
		response := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ns2:sendBillResponse xmlns:ns2="http://service.sunat.gob.pe">
      <applicationResponse>` + base64.StdEncoding.EncodeToString([]byte(rejectionCDR)) + `</applicationResponse>
    </ns2:sendBillResponse>
  </soap:Body>
</soap:Envelope>`
		
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Crear configuración SUNAT con el servidor mock
	sunatConfig := &models.SUNATConfig{
		Endpoint: server.URL,
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Crear paquete de prueba
	testPackage := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Enviar documento
	response, err := sunatService.SendDocument(testPackage)

	// Verificar que no hay error (SUNAT responde con CDR de rechazo)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	
	// Verificar que la respuesta indica rechazo
	assert.Contains(t, string(response.CDR), "0003")
	assert.Contains(t, string(response.CDR), "El Comprobante ingresado no cumple con el estandar establecido")
	assert.Equal(t, "RECHAZADO", response.Status)
}

// TestSUNATServiceSendDocumentError prueba el error de comunicación con SUNAT
func TestSUNATServiceSendDocumentError(t *testing.T) {
	// Crear servidor mock que simula error de SUNAT
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simular error 500 de SUNAT
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
	}))
	defer server.Close()

	// Crear configuración SUNAT con el servidor mock
	sunatConfig := &models.SUNATConfig{
		Endpoint: server.URL,
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Crear paquete de prueba
	testPackage := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Enviar documento
	response, err := sunatService.SendDocument(testPackage)

	// Verificar que hay error
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Error en respuesta de SUNAT")
}

// TestSUNATServiceBuildSOAPRequest prueba la construcción del SOAP request
func TestSUNATServiceBuildSOAPRequest(t *testing.T) {
	// Crear configuración SUNAT
	sunatConfig := &models.SUNATConfig{
		Endpoint: "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService",
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Crear paquete de prueba
	testPackage := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Construir SOAP request
	soapRequest, err := sunatService.buildSOAPRequest(testPackage)

	// Verificar que no hay error
	assert.NoError(t, err)
	assert.NotNil(t, soapRequest)

	// Verificar estructura SOAP
	soapStr := string(soapRequest)
	assert.Contains(t, soapStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, soapStr, "<soap:Envelope")
	assert.Contains(t, soapStr, "<soap:Header>")
	assert.Contains(t, soapStr, "<wsse:Security>")
	assert.Contains(t, soapStr, "<wsse:UsernameToken>")
	assert.Contains(t, soapStr, "<wsse:Username>20103129061MODDATOS</wsse:Username>")
	assert.Contains(t, soapStr, "<wsse:Password>MODDATOS</wsse:Password>")
	assert.Contains(t, soapStr, "<soap:Body>")
	assert.Contains(t, soapStr, "<sendBill")
	assert.Contains(t, soapStr, "<fileName>20123456789-01-F001-00000001.zip</fileName>")
	assert.Contains(t, soapStr, "<contentFile>")
	assert.Contains(t, soapStr, base64.StdEncoding.EncodeToString([]byte("test zip content")))
}

// TestSUNATServiceParseSOAPResponse prueba el parsing de respuestas SOAP
func TestSUNATServiceParseSOAPResponse(t *testing.T) {
	// Crear configuración SUNAT
	sunatConfig := &models.SUNATConfig{
		Endpoint: "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService",
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Respuesta SOAP de aceptación
	acceptanceResponse := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ns2:sendBillResponse xmlns:ns2="http://service.sunat.gob.pe">
      <applicationResponse>` + base64.StdEncoding.EncodeToString([]byte("CDR-ACEPTADO")) + `</applicationResponse>
    </ns2:sendBillResponse>
  </soap:Body>
</soap:Envelope>`

	// Parsear respuesta
	response, err := sunatService.parseSOAPResponse([]byte(acceptanceResponse))

	// Verificar que no hay error
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, string(response.CDR), "CDR-ACEPTADO")
	assert.Equal(t, "ACEPTADO", response.Status)

	// Respuesta SOAP de rechazo
	rejectionCDR := `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2">
  <cbc:ReferenceID>20123456789-01-F001-00000001</cbc:ReferenceID>
  <cbc:ResponseCode>0003</cbc:ResponseCode>
  <cbc:Description>El Comprobante ingresado no cumple con el estandar establecido.</cbc:Description>
</ApplicationResponse>`

	rejectionResponse := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ns2:sendBillResponse xmlns:ns2="http://service.sunat.gob.pe">
      <applicationResponse>` + base64.StdEncoding.EncodeToString([]byte(rejectionCDR)) + `</applicationResponse>
    </ns2:sendBillResponse>
  </soap:Body>
</soap:Envelope>`

	// Parsear respuesta de rechazo
	response, err = sunatService.parseSOAPResponse([]byte(rejectionResponse))

	// Verificar que no hay error
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, string(response.CDR), "0003")
	assert.Contains(t, string(response.CDR), "El Comprobante ingresado no cumple con el estandar establecido")
	assert.Equal(t, "RECHAZADO", response.Status)
}

// TestSUNATServiceValidatePackage prueba la validación del paquete antes del envío
func TestSUNATServiceValidatePackage(t *testing.T) {
	// Crear configuración SUNAT
	sunatConfig := &models.SUNATConfig{
		Endpoint: "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService",
		Username: "20103129061MODDATOS",
		Password: "MODDATOS",
		Timeout:  30,
	}

	// Crear servicio de codificación
	encodingService := NewEncodingService()
	
	// Crear servicio SUNAT
	sunatService := NewSUNATService(sunatConfig, encodingService)

	// Paquete válido
	validPackage := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Validar paquete válido
	err := sunatService.validatePackage(validPackage)
	assert.NoError(t, err)

	// Paquete inválido - sin nombre de archivo
	invalidPackage1 := &SUNATPackage{
		FileName:     "",
		ZipContent:   []byte("test zip content"),
		Base64Content: base64.StdEncoding.EncodeToString([]byte("test zip content")),
		XMLContent:   []byte("test xml content"),
	}

	// Validar paquete inválido
	err = sunatService.validatePackage(invalidPackage1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nombre de archivo requerido")

	// Paquete inválido - ZIP vacío
	invalidPackage2 := &SUNATPackage{
		FileName:     "20123456789-01-F001-00000001.zip",
		ZipContent:   []byte{},
		Base64Content: "",
		XMLContent:   []byte("test xml content"),
	}

	// Validar paquete inválido
	err = sunatService.validatePackage(invalidPackage2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contenido ZIP requerido")
} 