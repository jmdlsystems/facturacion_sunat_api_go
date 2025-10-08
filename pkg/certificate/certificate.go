package certificate

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/pkcs12"
)

type Manager struct{}

type CertificateInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	SerialNumber string    `json:"serial_number"`
	KeyUsage     []string  `json:"key_usage"`
	IsValid      bool      `json:"is_valid"`
	DaysUntilExpiry int    `json:"days_until_expiry"`
}

func NewManager() *Manager {
	return &Manager{}
}

// LoadCertificate carga un certificado PKCS#12 (.p12/.pfx)
func (m *Manager) LoadCertificate(certPath, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Leer archivo de certificado
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo archivo de certificado: %v", err)
	}

	// Decodificar PKCS#12
	privateKey, cert, err := pkcs12.Decode(certData, password)
	if err != nil {
		return nil, nil, fmt.Errorf("error decodificando certificado PKCS#12: %v", err)
	}

	// Verificar que sea una clave RSA
	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("la clave privada debe ser RSA")
	}

	return cert, rsaKey, nil
}

// LoadPEMCertificate carga un certificado en formato PEM
func (m *Manager) LoadPEMCertificate(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Cargar certificado
	certPEM, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo certificado PEM: %v", err)
	}

	// Decodificar certificado PEM
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("no se pudo decodificar el certificado PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error parseando certificado: %v", err)
	}

	// Cargar clave privada
	keyPEM, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo clave privada: %v", err)
	}

	// Decodificar clave privada PEM
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no se pudo decodificar la clave privada PEM")
	}

	var privateKey *rsa.PrivateKey
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("error parseando clave privada PKCS8: %v", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("la clave privada debe ser RSA")
		}
	default:
		return nil, nil, fmt.Errorf("tipo de clave privada no soportado: %s", keyBlock.Type)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error parseando clave privada: %v", err)
	}

	return cert, privateKey, nil
}

// LoadBase64Certificate carga un certificado desde archivos base64
func (m *Manager) LoadBase64Certificate(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Cargar certificado base64
	certBase64, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo archivo de certificado base64: %v", err)
	}

	// Decodificar certificado base64
	certData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(certBase64)))
	if err != nil {
		return nil, nil, fmt.Errorf("error decodificando certificado base64: %v", err)
	}

	// Decodificar PEM del certificado
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("no se pudo decodificar el certificado PEM")
	}

	// Parsear certificado
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error parseando certificado: %v", err)
	}

	// Cargar clave privada base64
	keyBase64, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo archivo de clave privada base64: %v", err)
	}

	// Decodificar clave privada base64
	keyData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyBase64)))
	if err != nil {
		return nil, nil, fmt.Errorf("error decodificando clave privada base64: %v", err)
	}

	// Decodificar PEM de la clave privada
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no se pudo decodificar la clave privada PEM")
	}

	// Parsear clave privada según el tipo
	var privateKey *rsa.PrivateKey
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("error parseando clave privada PKCS8: %v", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("la clave privada debe ser RSA")
		}
	default:
		return nil, nil, fmt.Errorf("tipo de clave privada no soportado: %s", keyBlock.Type)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error parseando clave privada: %v", err)
	}

	return cert, privateKey, nil
}

// ValidateCertificate valida un certificado para uso con SUNAT
func (m *Manager) ValidateCertificate(cert *x509.Certificate) error {
	now := time.Now()

	// Verificar vigencia
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificado aún no es válido (válido desde: %v)", cert.NotBefore)
	}

	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificado expirado (expiró: %v)", cert.NotAfter)
	}

	// Verificar que sea para firma digital
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("certificado no permite firma digital")
	}

	// Verificar algoritmo de clave pública
	if _, ok := cert.PublicKey.(*rsa.PublicKey); !ok {
		return fmt.Errorf("certificado debe usar clave RSA")
	}

	// Verificar longitud mínima de clave RSA
	rsaKey := cert.PublicKey.(*rsa.PublicKey)
	if rsaKey.Size() < 256 { // 2048 bits mínimo
		return fmt.Errorf("clave RSA debe ser de al menos 2048 bits")
	}

	return nil
}

// GetCertificateInfo extrae información detallada del certificado
func (m *Manager) GetCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	now := time.Now()
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	
	// Determinar usos de clave
	var keyUsage []string
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		keyUsage = append(keyUsage, "Digital Signature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		keyUsage = append(keyUsage, "Key Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		keyUsage = append(keyUsage, "Data Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		keyUsage = append(keyUsage, "Certificate Sign")
	}

	// Verificar si es válido
	isValid := now.After(cert.NotBefore) && now.Before(cert.NotAfter)

	return &CertificateInfo{
		Subject:         cert.Subject.String(),
		Issuer:          cert.Issuer.String(),
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		SerialNumber:    cert.SerialNumber.String(),
		KeyUsage:        keyUsage,
		IsValid:         isValid,
		DaysUntilExpiry: daysUntilExpiry,
	}
}

// VerifyCertificateChain verifica la cadena de certificados
func (m *Manager) VerifyCertificateChain(cert *x509.Certificate, intermediates []*x509.Certificate, roots *x509.CertPool) error {
	// Crear pool de certificados intermedios
	intermediatePool := x509.NewCertPool()
	for _, intermediate := range intermediates {
		intermediatePool.AddCert(intermediate)
	}

	// Configurar opciones de verificación
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediatePool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	// Verificar cadena
	_, err := cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("error verificando cadena de certificados: %v", err)
	}

	return nil
}

// CheckCertificateExpiry verifica si el certificado está próximo a expirar
func (m *Manager) CheckCertificateExpiry(cert *x509.Certificate, warningDays int) (bool, int, error) {
	now := time.Now()
	
	if now.After(cert.NotAfter) {
		return true, 0, fmt.Errorf("certificado expirado")
	}

	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	
	if daysUntilExpiry <= warningDays {
		return true, daysUntilExpiry, fmt.Errorf("certificado expira en %d días", daysUntilExpiry)
	}

	return false, daysUntilExpiry, nil
}

// ExtractPublicKeyInfo extrae información de la clave pública
func (m *Manager) ExtractPublicKeyInfo(cert *x509.Certificate) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	switch pubKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		info["algorithm"] = "RSA"
		info["key_size"] = pubKey.Size() * 8 // en bits
		info["modulus_size"] = pubKey.N.BitLen()
		info["exponent"] = pubKey.E
	default:
		return nil, fmt.Errorf("tipo de clave pública no soportado")
	}

	return info, nil
}

// GetCertificateFingerprint calcula el fingerprint del certificado
func (m *Manager) GetCertificateFingerprint(cert *x509.Certificate, algorithm string) (string, error) {
	switch algorithm {
	case "SHA1":
		hash := sha1.Sum(cert.Raw)
		return fmt.Sprintf("%x", hash), nil
	case "SHA256":
		hash := sha256.Sum256(cert.Raw)
		return fmt.Sprintf("%x", hash), nil
	default:
		return "", fmt.Errorf("algoritmo de fingerprint no soportado: %s", algorithm)
	}
}

// ValidateForSUNAT valida específicamente para requisitos SUNAT
func (m *Manager) ValidateForSUNAT(cert *x509.Certificate) error {
	// Validaciones básicas
	if err := m.ValidateCertificate(cert); err != nil {
		return err
	}

	// Verificar que tenga el OID correcto para firma digital
	hasDigitalSignature := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageCodeSigning || usage == x509.ExtKeyUsageEmailProtection {
			hasDigitalSignature = true
			break
		}
	}

	// Si no tiene ExtKeyUsage específico, verificar KeyUsage básico
	if !hasDigitalSignature && cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("certificado no válido para firma digital SUNAT")
	}

	// Verificar que el emisor sea una CA reconocida (opcional)
	// Esta validación puede ser más específica según los requisitos de SUNAT

	return nil
}

// ConvertToPEM convierte un certificado a formato PEM
func (m *Manager) ConvertToPEM(cert *x509.Certificate) ([]byte, error) {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(block), nil
}

// SaveCertificateInfo guarda información del certificado en un archivo JSON
func (m *Manager) SaveCertificateInfo(cert *x509.Certificate, outputPath string) error {
	// Obtener información completa del certificado
	info := m.GetCertificateInfo(cert)
	
	// Agregar información adicional técnica
	extendedInfo := make(map[string]interface{})
	
	// Información básica
	extendedInfo["basic_info"] = info
	
	// Información técnica de la clave pública
	publicKeyInfo, err := m.ExtractPublicKeyInfo(cert)
	if err == nil {
		extendedInfo["public_key_info"] = publicKeyInfo
	}
	
	// Fingerprints
	fingerprints := make(map[string]string)
	if sha1, err := m.GetCertificateFingerprint(cert, "SHA1"); err == nil {
		fingerprints["sha1"] = sha1
	}
	if sha256, err := m.GetCertificateFingerprint(cert, "SHA256"); err == nil {
		fingerprints["sha256"] = sha256
	}
	extendedInfo["fingerprints"] = fingerprints
	
	// Información de extensiones del certificado
	extensions := make([]map[string]interface{}, 0)
	for _, ext := range cert.Extensions {
		extInfo := map[string]interface{}{
			"id":       ext.Id.String(),
			"critical": ext.Critical,
			"value":    fmt.Sprintf("%x", ext.Value),
		}
		extensions = append(extensions, extInfo)
	}
	extendedInfo["extensions"] = extensions
	
	// Extended Key Usage
	extKeyUsage := make([]string, 0)
	for _, usage := range cert.ExtKeyUsage {
		switch usage {
		case x509.ExtKeyUsageAny:
			extKeyUsage = append(extKeyUsage, "Any")
		case x509.ExtKeyUsageServerAuth:
			extKeyUsage = append(extKeyUsage, "Server Authentication")
		case x509.ExtKeyUsageClientAuth:
			extKeyUsage = append(extKeyUsage, "Client Authentication")
		case x509.ExtKeyUsageCodeSigning:
			extKeyUsage = append(extKeyUsage, "Code Signing")
		case x509.ExtKeyUsageEmailProtection:
			extKeyUsage = append(extKeyUsage, "Email Protection")
		case x509.ExtKeyUsageTimeStamping:
			extKeyUsage = append(extKeyUsage, "Time Stamping")
		case x509.ExtKeyUsageOCSPSigning:
			extKeyUsage = append(extKeyUsage, "OCSP Signing")
		default:
			extKeyUsage = append(extKeyUsage, fmt.Sprintf("Unknown(%d)", usage))
		}
	}
	extendedInfo["extended_key_usage"] = extKeyUsage
	
	// Subject Alternative Names
	if len(cert.DNSNames) > 0 || len(cert.EmailAddresses) > 0 || len(cert.IPAddresses) > 0 {
		san := make(map[string]interface{})
		if len(cert.DNSNames) > 0 {
			san["dns_names"] = cert.DNSNames
		}
		if len(cert.EmailAddresses) > 0 {
			san["email_addresses"] = cert.EmailAddresses
		}
		if len(cert.IPAddresses) > 0 {
			ipStrings := make([]string, len(cert.IPAddresses))
			for i, ip := range cert.IPAddresses {
				ipStrings[i] = ip.String()
			}
			san["ip_addresses"] = ipStrings
		}
		extendedInfo["subject_alternative_names"] = san
	}
	
	// Información de validación SUNAT
	sunatValidation := make(map[string]interface{})
	sunatValidation["is_valid_for_sunat"] = m.ValidateForSUNAT(cert) == nil
	
	// Verificar próximo vencimiento (30 días)
	isExpiringSoon, daysLeft, expErr := m.CheckCertificateExpiry(cert, 30)
	sunatValidation["expiring_soon"] = isExpiringSoon
	sunatValidation["days_until_expiry"] = daysLeft
	if expErr != nil {
		sunatValidation["expiry_warning"] = expErr.Error()
	}
	
	extendedInfo["sunat_validation"] = sunatValidation
	
	// Información de la cadena de certificación
	chainInfo := make(map[string]interface{})
	chainInfo["is_self_signed"] = cert.Subject.String() == cert.Issuer.String()
	chainInfo["signature_algorithm"] = cert.SignatureAlgorithm.String()
	chainInfo["public_key_algorithm"] = cert.PublicKeyAlgorithm.String()
	extendedInfo["chain_info"] = chainInfo
	
	// Metadatos de generación del reporte
	metadata := map[string]interface{}{
		"generated_at":      time.Now().Format(time.RFC3339),
		"generator":         "SUNAT Certificate Manager",
		"version":           "1.0.0",
		"certificate_file":  filepath.Base(outputPath),
	}
	extendedInfo["metadata"] = metadata
	
	// Crear directorio si no existe
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %v", err)
	}
	
	// Serializar a JSON con formato legible
	jsonData, err := json.MarshalIndent(extendedInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando información del certificado: %v", err)
	}
	
	// Escribir archivo
	if err := ioutil.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo de información: %v", err)
	}
	
	return nil
}

// SaveCertificateReport genera un reporte detallado en múltiples formatos
func (m *Manager) SaveCertificateReport(cert *x509.Certificate, outputDir string, formats []string) error {
	// Crear directorio si no existe
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de reportes: %v", err)
	}
	
	baseFileName := fmt.Sprintf("certificate_report_%s", time.Now().Format("20060102_150405"))
	
	for _, format := range formats {
		switch format {
		case "json":
			jsonPath := filepath.Join(outputDir, baseFileName+".json")
			if err := m.SaveCertificateInfo(cert, jsonPath); err != nil {
				return fmt.Errorf("error guardando reporte JSON: %v", err)
			}
			
		case "txt":
			txtPath := filepath.Join(outputDir, baseFileName+".txt")
			if err := m.saveCertificateTextReport(cert, txtPath); err != nil {
				return fmt.Errorf("error guardando reporte TXT: %v", err)
			}
			
		case "pem":
			pemPath := filepath.Join(outputDir, baseFileName+".pem")
			pemData, err := m.ConvertToPEM(cert)
			if err != nil {
				return fmt.Errorf("error convirtiendo a PEM: %v", err)
			}
			if err := ioutil.WriteFile(pemPath, pemData, 0644); err != nil {
				return fmt.Errorf("error guardando PEM: %v", err)
			}
			
		default:
			return fmt.Errorf("formato no soportado: %s", format)
		}
	}
	
	return nil
}

// saveCertificateTextReport guarda un reporte en formato texto legible
func (m *Manager) saveCertificateTextReport(cert *x509.Certificate, outputPath string) error {
	info := m.GetCertificateInfo(cert)
	
	var report strings.Builder
	
	report.WriteString("REPORTE DE CERTIFICADO DIGITAL\n")
	report.WriteString("===============================\n\n")
	
	report.WriteString(fmt.Sprintf("Sujeto: %s\n", info.Subject))
	report.WriteString(fmt.Sprintf("Emisor: %s\n", info.Issuer))
	report.WriteString(fmt.Sprintf("Número de Serie: %s\n", info.SerialNumber))
	report.WriteString(fmt.Sprintf("Válido desde: %s\n", info.NotBefore.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Válido hasta: %s\n", info.NotAfter.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Estado: %s\n", func() string {
		if info.IsValid {
			return "VÁLIDO"
		}
		return "EXPIRADO/INVÁLIDO"
	}()))
	report.WriteString(fmt.Sprintf("Días hasta expiración: %d\n\n", info.DaysUntilExpiry))
	
	report.WriteString("Usos de Clave:\n")
	for _, usage := range info.KeyUsage {
		report.WriteString(fmt.Sprintf("  - %s\n", usage))
	}
	report.WriteString("\n")
	
	// Información técnica
	if pubKeyInfo, err := m.ExtractPublicKeyInfo(cert); err == nil {
		report.WriteString("Información de Clave Pública:\n")
		for key, value := range pubKeyInfo {
			report.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
		report.WriteString("\n")
	}
	
	// Fingerprints
	report.WriteString("Huellas Digitales:\n")
	if sha1, err := m.GetCertificateFingerprint(cert, "SHA1"); err == nil {
		report.WriteString(fmt.Sprintf("  SHA-1: %s\n", sha1))
	}
	if sha256, err := m.GetCertificateFingerprint(cert, "SHA256"); err == nil {
		report.WriteString(fmt.Sprintf("  SHA-256: %s\n", sha256))
	}
	report.WriteString("\n")
	
	// Validación SUNAT
	report.WriteString("Validación SUNAT:\n")
	if err := m.ValidateForSUNAT(cert); err == nil {
		report.WriteString("  ✓ Válido para SUNAT\n")
	} else {
		report.WriteString(fmt.Sprintf("  ✗ No válido para SUNAT: %s\n", err.Error()))
	}
	
	report.WriteString(fmt.Sprintf("\nReporte generado: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	
	return ioutil.WriteFile(outputPath, []byte(report.String()), 0644)
}