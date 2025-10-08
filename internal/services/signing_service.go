package services

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"facturacion_sunat_api_go/internal/models"
	"facturacion_sunat_api_go/pkg/certificate"
	"fmt"
	"strings"
	"time"
)

type SigningService struct {
	certificateManager *certificate.Manager
	ublService         *UBLService
}

func NewSigningService(certManager *certificate.Manager, ublService *UBLService) *SigningService {
	return &SigningService{
		certificateManager: certManager,
		ublService:         ublService,
	}
}

// SignDocument firma digitalmente un documento XML usando el certificado SUNAT
func (s *SigningService) SignDocument(xmlData []byte, certPath, certPassword string) ([]byte, error) {
	var cert *x509.Certificate
	var privateKey *rsa.PrivateKey
	var err error

	// Detectar tipo de certificado y cargarlo apropiadamente
	if strings.HasSuffix(certPath, ".b64") || strings.HasSuffix(certPath, ".base64") {
		// Es un archivo base64, necesitamos la ruta de la clave también
		var keyPath string
		if strings.HasSuffix(certPath, "/cert.b64") {
			keyPath = strings.Replace(certPath, "/cert.b64", "/key.b64", 1)
		} else if strings.HasSuffix(certPath, "\\cert.b64") {
			keyPath = strings.Replace(certPath, "\\cert.b64", "\\key.b64", 1)
		} else if strings.HasSuffix(certPath, "cert.b64") {
			keyPath = strings.Replace(certPath, "cert.b64", "key.b64", 1)
		} else {
			// Reemplazar solo el nombre del archivo (cert -> key)
			keyPath = strings.Replace(certPath, "cert", "key", 1)
		}
		cert, privateKey, err = s.certificateManager.LoadBase64Certificate(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("error cargando certificado base64: %v", err)
		}
	} else if strings.HasSuffix(certPath, ".pem") {
		// Es un archivo PEM, usar LoadPEMCertificate
		var keyPath string
		if strings.HasSuffix(certPath, "/cert.pem") {
			keyPath = strings.Replace(certPath, "/cert.pem", "/key.pem", 1)
		} else if strings.HasSuffix(certPath, "\\cert.pem") {
			keyPath = strings.Replace(certPath, "\\cert.pem", "\\key.pem", 1)
		} else if strings.HasSuffix(certPath, "cert.pem") {
			keyPath = strings.Replace(certPath, "cert.pem", "key.pem", 1)
		} else {
			keyPath = strings.Replace(certPath, "cert", "key", 1)
		}
		cert, privateKey, err = s.certificateManager.LoadPEMCertificate(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("error cargando certificado PEM: %v", err)
		}
	} else {
		// Es un archivo PKCS#12 tradicional
		cert, privateKey, err = s.certificateManager.LoadCertificate(certPath, certPassword)
		if err != nil {
			return nil, fmt.Errorf("error cargando certificado: %v", err)
		}
	}

	// Generar XML canónico
	canonicalXML, err := s.ublService.GenerateCanonicalXML(xmlData)
	if err != nil {
		return nil, fmt.Errorf("error generando XML canónico: %v", err)
	}

	// Calcular hash SHA-256
	hash := sha256.Sum256(canonicalXML)
	digestValue := base64.StdEncoding.EncodeToString(hash[:])

	// Crear SignedInfo
	signedInfo := s.createSignedInfo(digestValue)

	// Serializar SignedInfo para firmado
	signedInfoXML, err := xml.Marshal(signedInfo)
	if err != nil {
		return nil, fmt.Errorf("error serializando SignedInfo: %v", err)
	}

	// Canonicalizar SignedInfo
	canonicalSignedInfo, err := s.canonicalizeSignedInfo(signedInfoXML)
	if err != nil {
		return nil, fmt.Errorf("error canonicalizando SignedInfo: %v", err)
	}

	// Firmar SignedInfo
	signedInfoHash := sha256.Sum256(canonicalSignedInfo)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, signedInfoHash[:])
	if err != nil {
		return nil, fmt.Errorf("error firmando documento: %v", err)
	}

	signatureValue := base64.StdEncoding.EncodeToString(signature)

	// Obtener certificado en formato base64
	certBytes := cert.Raw
	certBase64 := base64.StdEncoding.EncodeToString(certBytes)

	// Crear estructura de firma digital
	digitalSignature := s.createDigitalSignature(signedInfo, signatureValue, certBase64)

	// Agregar firma al documento XML
	signedXML, err := s.ublService.AddDigitalSignature(xmlData, digitalSignature)
	if err != nil {
		return nil, fmt.Errorf("error agregando firma al documento: %v", err)
	}

	return signedXML, nil
}

// createSignedInfo crea la estructura SignedInfo para la firma según estándares XMLDSig
func (s *SigningService) createSignedInfo(digestValue string) *models.SignedInfo {
	return &models.SignedInfo{
		CanonicalizationMethod: &models.CanonicalizationMethod{
			Algorithm: "http://www.w3.org/TR/2001/REC-xml-c14n-20010315",
		},
		SignatureMethod: &models.SignatureMethod{
			Algorithm: "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
		},
		Reference: &models.Reference{
			URI: "",
			Transforms: &models.Transforms{
				Transform: []models.Transform{
					{
						Algorithm: "http://www.w3.org/2000/09/xmldsig#enveloped-signature",
					},
					{
						Algorithm: "http://www.w3.org/TR/2001/REC-xml-c14n-20010315",
					},
				},
			},
			DigestMethod: &models.DigestMethod{
				Algorithm: "http://www.w3.org/2001/04/xmlenc#sha256",
			},
			DigestValue: digestValue,
		},
	}
}

// createDigitalSignature crea la estructura completa de firma digital según XMLDSig
func (s *SigningService) createDigitalSignature(signedInfo *models.SignedInfo, signatureValue, certBase64 string) *models.DigitalSignature {
	return &models.DigitalSignature{
		Id: "SignatureElement",
		SignedInfo: signedInfo,
		SignatureValue: &models.SignatureValue{
			Value: signatureValue,
		},
		KeyInfo: &models.KeyInfo{
			X509Data: &models.X509Data{
				X509Certificate: certBase64,
			},
		},
	}
}

// canonicalizeSignedInfo canonicaliza el SignedInfo según C14N XMLDSig
func (s *SigningService) canonicalizeSignedInfo(signedInfoXML []byte) ([]byte, error) {
	// Implementación de canonicalización C14N según XMLDSig
	xmlStr := string(signedInfoXML)
	
	// Remover declaración XML si existe
	if strings.HasPrefix(xmlStr, "<?xml") {
		xmlDeclarationEnd := strings.Index(xmlStr, "?>")
		if xmlDeclarationEnd != -1 {
			xmlStr = xmlStr[xmlDeclarationEnd+2:]
		}
	}
	
	// Normalizar espacios en blanco
	xmlStr = strings.TrimSpace(xmlStr)
	
	// Remover espacios en blanco innecesarios entre elementos
	xmlStr = strings.ReplaceAll(xmlStr, ">\n", ">")
	xmlStr = strings.ReplaceAll(xmlStr, "\n<", "<")
	xmlStr = strings.ReplaceAll(xmlStr, ">  <", "><")
	xmlStr = strings.ReplaceAll(xmlStr, "> <", "><")
	xmlStr = strings.ReplaceAll(xmlStr, ">\t<", "><")
	
	// Normalizar atributos (ordenar alfabéticamente)
	// Esta es una implementación simplificada
	// En producción se debería usar una librería de canonicalización completa
	
	return []byte(xmlStr), nil
}

// VerifySignature verifica la firma digital de un documento
func (s *SigningService) VerifySignature(signedXML []byte) (bool, error) {
	// Extraer la firma del documento
	signature, err := s.extractSignature(signedXML)
	if err != nil {
		return false, fmt.Errorf("error extrayendo firma: %v", err)
	}

	// Extraer el certificado
	certData, err := base64.StdEncoding.DecodeString(signature.KeyInfo.X509Data.X509Certificate)
	if err != nil {
		return false, fmt.Errorf("error decodificando certificado: %v", err)
	}

	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return false, fmt.Errorf("error parseando certificado: %v", err)
	}

	// Verificar que el certificado no haya expirado
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return false, fmt.Errorf("certificado expirado o no válido aún")
	}

	// Extraer clave pública
	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("clave pública no es RSA")
	}

	// Recrear el documento sin la firma para verificación
	docWithoutSignature, err := s.removeSignature(signedXML)
	if err != nil {
		return false, fmt.Errorf("error removiendo firma para verificación: %v", err)
	}

	// Canonicalizar documento
	canonicalDoc, err := s.ublService.GenerateCanonicalXML(docWithoutSignature)
	if err != nil {
		return false, fmt.Errorf("error canonicalizando documento: %v", err)
	}

	// Calcular hash del documento
	docHash := sha256.Sum256(canonicalDoc)
	expectedDigest := base64.StdEncoding.EncodeToString(docHash[:])

	// Verificar que el digest coincida
	if signature.SignedInfo.Reference.DigestValue != expectedDigest {
		return false, fmt.Errorf("digest del documento no coincide")
	}

	// Recrear SignedInfo y verificar firma
	signedInfoXML, err := xml.Marshal(signature.SignedInfo)
	if err != nil {
		return false, fmt.Errorf("error serializando SignedInfo: %v", err)
	}

	canonicalSignedInfo, err := s.canonicalizeSignedInfo(signedInfoXML)
	if err != nil {
		return false, fmt.Errorf("error canonicalizando SignedInfo: %v", err)
	}

	signedInfoHash := sha256.Sum256(canonicalSignedInfo)

	// Decodificar firma
	signatureBytes, err := base64.StdEncoding.DecodeString(signature.SignatureValue.Value)
	if err != nil {
		return false, fmt.Errorf("error decodificando firma: %v", err)
	}

	// Verificar firma
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, signedInfoHash[:], signatureBytes)
	if err != nil {
		return false, fmt.Errorf("verificación de firma falló: %v", err)
	}

	return true, nil
}

// extractSignature extrae la estructura de firma del XML
func (s *SigningService) extractSignature(xmlData []byte) (*models.DigitalSignature, error) {
	xmlStr := string(xmlData)
	
	// Buscar el elemento Signature
	startSignature := strings.Index(xmlStr, "<ds:Signature")
	if startSignature == -1 {
		return nil, fmt.Errorf("firma digital no encontrada")
	}
	
	endSignature := strings.Index(xmlStr[startSignature:], "</ds:Signature>")
	if endSignature == -1 {
		return nil, fmt.Errorf("cierre de firma digital no encontrado")
	}
	
	signatureXML := xmlStr[startSignature : startSignature+endSignature+len("</ds:Signature>")]
	
	var signature models.DigitalSignature
	if err := xml.Unmarshal([]byte(signatureXML), &signature); err != nil {
		return nil, fmt.Errorf("error parseando firma digital: %v", err)
	}
	
	return &signature, nil
}

// removeSignature remueve la firma del documento para verificación
func (s *SigningService) removeSignature(xmlData []byte) ([]byte, error) {
	xmlStr := string(xmlData)
	
	// Buscar y remover el elemento de firma
	startSignature := strings.Index(xmlStr, "<ds:Signature")
	if startSignature == -1 {
		return xmlData, nil // No hay firma
	}
	
	endSignature := strings.Index(xmlStr[startSignature:], "</ds:Signature>")
	if endSignature == -1 {
		return nil, fmt.Errorf("estructura de firma inválida")
	}
	
	endSignature += startSignature + len("</ds:Signature>")
	
	// Remover la firma
	docWithoutSignature := xmlStr[:startSignature] + xmlStr[endSignature:]
	
	return []byte(docWithoutSignature), nil
}

// GetCertificateInfo extrae información del certificado
func (s *SigningService) GetCertificateInfo(certPath, certPassword string) (*certificate.CertificateInfo, error) {
	cert, _, err := s.certificateManager.LoadCertificate(certPath, certPassword)
	if err != nil {
		return nil, fmt.Errorf("error cargando certificado: %v", err)
	}

	return &certificate.CertificateInfo{
		Subject:    cert.Subject.String(),
		Issuer:     cert.Issuer.String(),
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		SerialNumber: cert.SerialNumber.String(),
	}, nil
}

// ValidateCertificate valida que el certificado sea válido para SUNAT
func (s *SigningService) ValidateCertificate(certPath, certPassword string) error {
	cert, _, err := s.certificateManager.LoadCertificate(certPath, certPassword)
	if err != nil {
		return fmt.Errorf("error cargando certificado: %v", err)
	}

	// Verificar que no haya expirado
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificado aún no es válido")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificado expirado")
	}

	// Verificar que sea un certificado RSA
	if _, ok := cert.PublicKey.(*rsa.PublicKey); !ok {
		return fmt.Errorf("certificado debe usar clave RSA")
	}

	// Verificar propósito del certificado (debe permitir firma digital)
	if (cert.KeyUsage & x509.KeyUsageDigitalSignature) == 0 {
		return fmt.Errorf("certificado no permite firma digital")
	}

	return nil
}

// CreateSignatureReference crea una referencia para la firma
func (s *SigningService) CreateSignatureReference() *models.Signature {
	return &models.Signature{
		ID: "SignatureSP",
		SignatoryParty: &models.SignatoryParty{
			PartyIdentification: &models.PartyIdentification{
				ID: &models.ID{
					Value: "20123456789",
				},
			},
			PartyName: &models.PartyName{
				Name: "SUNAT",
			},
		},
		DigitalSignatureAttachment: &models.DigitalSignatureAttachment{
			ExternalReference: &models.ExternalReference{
				URI: "#SignatureElement",
			},
		},
	}
}