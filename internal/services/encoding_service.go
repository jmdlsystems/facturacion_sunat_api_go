package services

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type EncodingService struct{}

func NewEncodingService() *EncodingService {
	return &EncodingService{}
}

// EncodeToBase64 codifica datos a Base64
func (s *EncodingService) EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeFromBase64 decodifica datos desde Base64
func (s *EncodingService) DecodeFromBase64(encodedData string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("error decodificando Base64: %v", err)
	}
	return decoded, nil
}

// CreateZipFile crea un archivo ZIP con el XML firmado
func (s *EncodingService) CreateZipFile(xmlData []byte, fileName string) ([]byte, error) {
	// Validar datos de entrada
	if len(xmlData) == 0 {
		return nil, fmt.Errorf("datos XML vacíos")
	}

	// Crear buffer para el ZIP
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// Asegurar extensión .xml
	if !strings.HasSuffix(fileName, ".xml") {
		fileName += ".xml"
	}

	// Crear archivo dentro del ZIP
	fileWriter, err := zipWriter.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("error creando archivo en ZIP: %v", err)
	}

	// Escribir contenido XML
	bytesWritten, err := fileWriter.Write(xmlData)
	if err != nil {
		return nil, fmt.Errorf("error escribiendo XML en ZIP: %v", err)
	}

	if bytesWritten != len(xmlData) {
		return nil, fmt.Errorf("no se escribieron todos los bytes del XML: %d de %d", bytesWritten, len(xmlData))
	}

	// Cerrar ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error cerrando ZIP: %v", err)
	}

	zipData := zipBuffer.Bytes()
	if len(zipData) == 0 {
		return nil, fmt.Errorf("ZIP resultante está vacío")
	}

	return zipData, nil
}

// ExtractFromZip extrae archivos de un ZIP
func (s *EncodingService) ExtractFromZip(zipData []byte) (map[string][]byte, error) {
	// Crear reader desde bytes
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("error abriendo ZIP: %v", err)
	}

	extractedFiles := make(map[string][]byte)

	// Extraer cada archivo
	for _, file := range zipReader.File {
		// Abrir archivo
		fileReader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("error abriendo archivo %s: %v", file.Name, err)
		}

		// Leer contenido
		content, err := io.ReadAll(fileReader)
		if err != nil {
			fileReader.Close()
			return nil, fmt.Errorf("error leyendo archivo %s: %v", file.Name, err)
		}

		extractedFiles[file.Name] = content
		fileReader.Close()
	}

	return extractedFiles, nil
}

// ProcessForSUNAT procesa el XML firmado para envío a SUNAT
func (s *EncodingService) ProcessForSUNAT(signedXML []byte, documentID string) (*SUNATPackage, error) {
	// Validar datos de entrada
	if len(signedXML) == 0 {
		return nil, fmt.Errorf("XML firmado está vacío")
	}

	if documentID == "" {
		return nil, fmt.Errorf("ID de documento requerido")
	}

	// Generar nombre de archivo según estándar SUNAT
	fileName := s.generateSUNATFileName(documentID)

	// Crear ZIP con el XML firmado
	zipData, err := s.CreateZipFile(signedXML, fileName)
	if err != nil {
		return nil, fmt.Errorf("error creando ZIP para SUNAT: %v", err)
	}

	// Validar que el ZIP se creó correctamente
	if len(zipData) == 0 {
		return nil, fmt.Errorf("ZIP creado está vacío")
	}

	// Codificar ZIP a Base64
	base64Data := s.EncodeToBase64(zipData)

	// Validar que el Base64 se generó correctamente
	if base64Data == "" {
		return nil, fmt.Errorf("Base64 generado está vacío")
	}

	pkg := &SUNATPackage{
		FileName:     fileName + ".zip",
		ZipContent:   zipData,
		Base64Content: base64Data,
		XMLContent:   signedXML,
	}

	// Validar el paquete antes de retornarlo
	if err := pkg.ValidatePackage(); err != nil {
		return nil, fmt.Errorf("paquete generado es inválido: %v", err)
	}

	return pkg, nil
}

// generateSUNATFileName genera el nombre de archivo según convención SUNAT
func (s *EncodingService) generateSUNATFileName(documentID string) string {
	// Formato SUNAT: RUC-TIPODOC-SERIE-NUMERO
	// Ejemplo: 20123456789-01-F001-00000001
	// Validar formato
	if len(documentID) < 10 {
		return documentID
	}
	
	// Asegurar que tenga el formato correcto
	parts := strings.Split(documentID, "-")
	if len(parts) >= 4 {
		// Formato ya correcto
		return documentID
	}
	
	// Si no tiene el formato correcto, retornar como está
	return documentID
}

// ValidateZipStructure valida la estructura del ZIP para SUNAT
func (s *EncodingService) ValidateZipStructure(zipData []byte) error {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("ZIP inválido: %v", err)
	}

	if len(zipReader.File) == 0 {
		return fmt.Errorf("ZIP vacío")
	}

	if len(zipReader.File) > 1 {
		return fmt.Errorf("ZIP debe contener solo un archivo XML")
	}

	file := zipReader.File[0]
	if !strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
		return fmt.Errorf("archivo debe tener extensión .xml")
	}

	// Verificar tamaño del archivo
	if file.UncompressedSize64 > 10*1024*1024 { // 10MB máximo
		return fmt.Errorf("archivo XML muy grande (máximo 10MB)")
	}

	return nil
}

// CreateResponsePackage crea un paquete para procesar respuestas de SUNAT
func (s *EncodingService) CreateResponsePackage(responseData []byte) (*SUNATResponse, error) {
	// Determinar tipo de respuesta
	if s.isZipData(responseData) {
		// Respuesta en ZIP
		extractedFiles, err := s.ExtractFromZip(responseData)
		if err != nil {
			return nil, fmt.Errorf("error extrayendo respuesta ZIP: %v", err)
		}

		return &SUNATResponse{
			Type:         "ZIP",
			RawData:      responseData,
			ExtractedFiles: extractedFiles,
		}, nil
	}

	// Respuesta en XML directo
	return &SUNATResponse{
		Type:    "XML",
		RawData: responseData,
		XMLContent: responseData,
	}, nil
}

// isZipData verifica si los datos son un archivo ZIP
func (s *EncodingService) isZipData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Verificar magic number del ZIP (PK)
	return data[0] == 0x50 && data[1] == 0x4B
}

// CompressXML comprime XML usando mejor algoritmo
func (s *EncodingService) CompressXML(xmlData []byte, compressionLevel int) ([]byte, error) {
	var compressedBuffer bytes.Buffer
	
	// Crear ZIP writer con nivel de compresión
	zipWriter := zip.NewWriter(&compressedBuffer)
	
	// Configurar método de compresión
	writer, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:     "document.xml",
		Method:   zip.Deflate,
		Modified: time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("error configurando compresión: %v", err)
	}

	// Escribir datos comprimidos
	_, err = writer.Write(xmlData)
	if err != nil {
		return nil, fmt.Errorf("error comprimiendo datos: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error finalizando compresión: %v", err)
	}

	return compressedBuffer.Bytes(), nil
}

// ValidateBase64 valida que una cadena sea Base64 válido
func (s *EncodingService) ValidateBase64(encodedData string) error {
	// Remover espacios en blanco
	cleanData := strings.ReplaceAll(encodedData, " ", "")
	cleanData = strings.ReplaceAll(cleanData, "\n", "")
	cleanData = strings.ReplaceAll(cleanData, "\r", "")

	// Verificar longitud (debe ser múltiplo de 4)
	if len(cleanData)%4 != 0 {
		return fmt.Errorf("longitud de Base64 inválida")
	}

	// Intentar decodificar
	_, err := base64.StdEncoding.DecodeString(cleanData)
	if err != nil {
		return fmt.Errorf("Base64 inválido: %v", err)
	}

	return nil
}

// GetFileExtensionFromZip obtiene la extensión del archivo dentro del ZIP
func (s *EncodingService) GetFileExtensionFromZip(zipData []byte) (string, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", fmt.Errorf("error leyendo ZIP: %v", err)
	}

	if len(zipReader.File) == 0 {
		return "", fmt.Errorf("ZIP vacío")
	}

	fileName := zipReader.File[0].Name
	ext := filepath.Ext(fileName)
	return ext, nil
}

// CreateMultiFileZip crea un ZIP con múltiples archivos
func (s *EncodingService) CreateMultiFileZip(files map[string][]byte) ([]byte, error) {
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	for fileName, content := range files {
		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			return nil, fmt.Errorf("error creando archivo %s en ZIP: %v", fileName, err)
		}

		_, err = fileWriter.Write(content)
		if err != nil {
			return nil, fmt.Errorf("error escribiendo archivo %s: %v", fileName, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error cerrando ZIP múltiple: %v", err)
	}

	return zipBuffer.Bytes(), nil
}

// CalculateZipSize calcula el tamaño del ZIP resultante
func (s *EncodingService) CalculateZipSize(xmlData []byte) (int64, error) {
	zipData, err := s.CreateZipFile(xmlData, "temp.xml")
	if err != nil {
		return 0, fmt.Errorf("error calculando tamaño ZIP: %v", err)
	}
	return int64(len(zipData)), nil
}

// SUNATPackage representa un paquete preparado para SUNAT
type SUNATPackage struct {
	FileName      string `json:"file_name"`
	ZipContent    []byte `json:"-"`
	Base64Content string `json:"base64_content"`
	XMLContent    []byte `json:"-"`
	Size          int64  `json:"size"`
}

// SUNATResponse representa una respuesta de SUNAT
type SUNATResponse struct {
	Type           string            `json:"type"`
	RawData        []byte            `json:"-"`
	XMLContent     []byte            `json:"-"`
	ExtractedFiles map[string][]byte `json:"-"`
	ProcessedAt    time.Time         `json:"processed_at"`
}

// GetPackageInfo obtiene información del paquete
func (pkg *SUNATPackage) GetPackageInfo() map[string]interface{} {
	return map[string]interface{}{
		"file_name":      pkg.FileName,
		"zip_size":       len(pkg.ZipContent),
		"xml_size":       len(pkg.XMLContent),
		"base64_size":    len(pkg.Base64Content),
		"compression_ratio": float64(len(pkg.ZipContent)) / float64(len(pkg.XMLContent)),
	}
}

// ValidatePackage valida el paquete antes del envío
func (pkg *SUNATPackage) ValidatePackage() error {
	if pkg.FileName == "" {
		return fmt.Errorf("nombre de archivo requerido")
	}

	if len(pkg.ZipContent) == 0 {
		return fmt.Errorf("contenido ZIP vacío (tamaño: %d bytes)", len(pkg.ZipContent))
	}

	if len(pkg.XMLContent) == 0 {
		return fmt.Errorf("contenido XML vacío (tamaño: %d bytes)", len(pkg.XMLContent))
	}

	if pkg.Base64Content == "" {
		return fmt.Errorf("contenido Base64 vacío (longitud: %d caracteres)", len(pkg.Base64Content))
	}

	// Verificar que el Base64 sea válido
	encodingService := &EncodingService{}
	if err := encodingService.ValidateBase64(pkg.Base64Content); err != nil {
		return fmt.Errorf("Base64 inválido: %v", err)
	}

	// Verificar que el ZIP se puede leer correctamente
	if err := encodingService.ValidateZipStructure(pkg.ZipContent); err != nil {
		return fmt.Errorf("estructura ZIP inválida: %v", err)
	}

	return nil
}