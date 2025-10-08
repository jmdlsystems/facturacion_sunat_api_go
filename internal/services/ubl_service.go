package services

import (
	"encoding/xml"
	"facturacion_sunat_api_go/internal/models"
	"fmt"
	"strings"
)

type UBLService struct{}

func NewUBLService() *UBLService {
	return &UBLService{}
}

// SerializeToXML convierte una estructura UBL a XML
func (s *UBLService) SerializeToXML(ublDocument interface{}) ([]byte, error) {
	xmlData, err := xml.MarshalIndent(ublDocument, "", "	")
	if err != nil {
		return nil, fmt.Errorf("error marshaling UBL to XML: %v", err)
	}

	// Agregar declaración XML con encoding UTF-8
	xmlWithDeclaration := append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`), xmlData...)

	return xmlWithDeclaration, nil
}

// ValidateUBLStructure valida la estructura UBL antes de la serialización
func (s *UBLService) ValidateUBLStructure(ublDocument interface{}) error {
	switch doc := ublDocument.(type) {
	case *models.UBLInvoice:
		return s.validateInvoice(doc)
	case *models.UBLCreditNote:
		return s.validateCreditNote(doc)
	default:
		return fmt.Errorf("tipo de documento UBL no soportado")
	}
}

func (s *UBLService) validateInvoice(invoice *models.UBLInvoice) error {
	if invoice.ID == "" {
		return fmt.Errorf("ID de factura es requerido")
	}
	
	if invoice.IssueDate == "" {
		return fmt.Errorf("fecha de emisión es requerida")
	}

	if invoice.InvoiceTypeCode == "" {
		return fmt.Errorf("tipo de comprobante es requerido")
	}

	if invoice.DocumentCurrencyCode == "" {
		return fmt.Errorf("código de moneda es requerido")
	}

	if invoice.AccountingSupplierParty == nil {
		return fmt.Errorf("emisor es requerido")
	}

	if invoice.AccountingCustomerParty == nil {
		return fmt.Errorf("receptor es requerido")
	}

	if invoice.LegalMonetaryTotal == nil {
		return fmt.Errorf("totales monetarios son requeridos")
	}

	if len(invoice.InvoiceLines) == 0 {
		return fmt.Errorf("al menos una línea de factura es requerida")
	}

	// Validar líneas de factura
	for i, line := range invoice.InvoiceLines {
		if err := s.validateInvoiceLine(&line, i+1); err != nil {
			return fmt.Errorf("error en línea %d: %v", i+1, err)
		}
	}

	return nil
}

func (s *UBLService) validateCreditNote(creditNote *models.UBLCreditNote) error {
	if creditNote.ID == "" {
		return fmt.Errorf("ID de nota de crédito es requerido")
	}
	
	if creditNote.IssueDate == "" {
		return fmt.Errorf("fecha de emisión es requerida")
	}

	if creditNote.CreditNoteTypeCode == "" {
		return fmt.Errorf("tipo de nota de crédito es requerido")
	}

	if creditNote.DocumentCurrencyCode == "" {
		return fmt.Errorf("código de moneda es requerido")
	}

	if creditNote.AccountingSupplierParty == nil {
		return fmt.Errorf("emisor es requerido")
	}

	if creditNote.AccountingCustomerParty == nil {
		return fmt.Errorf("receptor es requerido")
	}

	if creditNote.LegalMonetaryTotal == nil {
		return fmt.Errorf("totales monetarios son requeridos")
	}

	if len(creditNote.CreditNoteLines) == 0 {
		return fmt.Errorf("al menos una línea de nota de crédito es requerida")
	}

	return nil
}

func (s *UBLService) validateInvoiceLine(line *models.InvoiceLine, lineNumber int) error {
	if line.ID == "" {
		return fmt.Errorf("ID de línea es requerido")
	}

	if line.InvoicedQuantity == nil {
		return fmt.Errorf("cantidad facturada es requerida")
	}

	if line.InvoicedQuantity.Value <= 0 {
		return fmt.Errorf("cantidad debe ser mayor a 0")
	}

	if line.LineExtensionAmount == nil {
		return fmt.Errorf("monto de línea es requerido")
	}

	if line.Item == nil {
		return fmt.Errorf("item es requerido")
	}

	if len(line.Item.Description) == 0 {
		return fmt.Errorf("descripción del item es requerida")
	}

	if line.Price == nil {
		return fmt.Errorf("precio es requerido")
	}

	return nil
}

// FormatXMLForSUNAT formatea el XML según los requerimientos de SUNAT
func (s *UBLService) FormatXMLForSUNAT(xmlData []byte) ([]byte, error) {
	// Convertir a string para manipulación
	xmlStr := string(xmlData)

	// Remover espacios en blanco innecesarios entre elementos
	xmlStr = strings.ReplaceAll(xmlStr, ">\n  <", "><")
	xmlStr = strings.ReplaceAll(xmlStr, ">\n<", "><")

	// Asegurar codificación UTF-8
	if !strings.Contains(xmlStr, "encoding=\"UTF-8\"") {
		xmlStr = strings.Replace(xmlStr, "<?xml version=\"1.0\"?>", 
			"<?xml version=\"1.0\" encoding=\"UTF-8\"?>", 1)
	}

	return []byte(xmlStr), nil
}

// AddDigitalSignature agrega la firma digital al documento UBL
func (s *UBLService) AddDigitalSignature(xmlData []byte, signature *models.DigitalSignature) ([]byte, error) {
	xmlStr := string(xmlData)

	// Buscar la posición donde insertar la extensión UBL DENTRO del elemento raíz
	// Primero buscar el elemento raíz (Invoice, CreditNote, etc.)
	rootElements := []string{"<Invoice", "<CreditNote", "<DebitNote"}
	var rootStartPos int = -1
	
	for _, element := range rootElements {
		pos := strings.Index(xmlStr, element)
		if pos != -1 {
			rootStartPos = pos
			break
		}
	}
	
	if rootStartPos == -1 {
		return nil, fmt.Errorf("no se encontró elemento raíz válido (Invoice, CreditNote, DebitNote)")
	}

	// Buscar el cierre del tag del elemento raíz
	rootTagEndPos := strings.Index(xmlStr[rootStartPos:], ">")
	if rootTagEndPos == -1 {
		return nil, fmt.Errorf("formato XML inválido: no se encontró cierre del tag raíz")
	}
	rootTagEndPos += rootStartPos + 1

	// Buscar UBLExtensions dentro del elemento raíz
	extensionPos := strings.Index(xmlStr[rootTagEndPos:], "<ext:UBLExtensions>")
	if extensionPos == -1 {
		// Si no existe UBLExtensions, crearlo DESPUÉS del tag de apertura del elemento raíz
		// pero ANTES del primer elemento hijo
		
		// Buscar el primer elemento hijo después del tag raíz
		firstChildPos := -1
		childElements := []string{"<cbc:UBLVersionID>", "<cbc:CustomizationID>", "<cbc:ID>"}
		for _, child := range childElements {
			pos := strings.Index(xmlStr[rootTagEndPos:], child)
			if pos != -1 && (firstChildPos == -1 || pos < firstChildPos) {
				firstChildPos = pos
			}
		}
		
		if firstChildPos == -1 {
			return nil, fmt.Errorf("no se encontró elemento hijo válido para insertar UBLExtensions")
		}
		
		// Crear la estructura UBLExtensions
		extensionsXML := `	<ext:UBLExtensions>
		<ext:UBLExtension>
			<ext:ExtensionContent>
			</ext:ExtensionContent>
		</ext:UBLExtension>
	</ext:UBLExtensions>
	`
		
		insertPos := rootTagEndPos + firstChildPos
		xmlStr = xmlStr[:insertPos] + extensionsXML + xmlStr[insertPos:]
		extensionPos = strings.Index(xmlStr, "<ext:ExtensionContent>")
	} else {
		extensionPos = rootTagEndPos + extensionPos + strings.Index(xmlStr[rootTagEndPos+extensionPos:], "<ext:ExtensionContent>")
	}

	if extensionPos == -1 {
		return nil, fmt.Errorf("no se pudo encontrar ExtensionContent")
	}

	// Serializar la firma digital
	signatureXML, err := xml.MarshalIndent(signature, "				", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializando firma digital: %v", err)
	}

	// Insertar la firma en ExtensionContent
	contentEndPos := strings.Index(xmlStr[extensionPos:], "</ext:ExtensionContent>")
	if contentEndPos == -1 {
		return nil, fmt.Errorf("no se pudo encontrar el cierre de ExtensionContent")
	}

	contentEndPos += extensionPos
	xmlStr = xmlStr[:extensionPos+len("<ext:ExtensionContent>")] + 
		"\n" + string(signatureXML) + "\n			" + 
		xmlStr[contentEndPos:]

	return []byte(xmlStr), nil
}

// GenerateCanonicalXML genera el XML canónico para la firma digital según C14N
func (s *UBLService) GenerateCanonicalXML(xmlData []byte) ([]byte, error) {
	// Implementación de canonicalización C14N según XMLDSig
	xmlStr := string(xmlData)

	// Remover declaración XML
	if strings.HasPrefix(xmlStr, "<?xml") {
		xmlDeclarationEnd := strings.Index(xmlStr, "?>")
		if xmlDeclarationEnd != -1 {
			xmlStr = xmlStr[xmlDeclarationEnd+2:]
		}
	}

	// Normalizar espacios en blanco
	xmlStr = strings.TrimSpace(xmlStr)
	
	// Remover comentarios XML
	for {
		startComment := strings.Index(xmlStr, "<!--")
		if startComment == -1 {
			break
		}
		endComment := strings.Index(xmlStr[startComment:], "-->")
		if endComment == -1 {
			break
		}
		xmlStr = xmlStr[:startComment] + xmlStr[startComment+endComment+3:]
	}

	// Remover espacios en blanco innecesarios entre elementos
	xmlStr = strings.ReplaceAll(xmlStr, ">\n", ">")
	xmlStr = strings.ReplaceAll(xmlStr, "\n<", "<")
	xmlStr = strings.ReplaceAll(xmlStr, ">  <", "><")
	xmlStr = strings.ReplaceAll(xmlStr, "> <", "><")
	xmlStr = strings.ReplaceAll(xmlStr, ">\t<", "><")
	
	// Normalizar espacios en blanco dentro de elementos de texto
	// Esta es una implementación simplificada
	// En producción se debería usar una librería de canonicalización completa como golang.org/x/net/html

	return []byte(xmlStr), nil
}

// ExtractElementForSigning extrae un elemento específico para firmado
func (s *UBLService) ExtractElementForSigning(xmlData []byte, elementName string) ([]byte, error) {
	xmlStr := string(xmlData)
	
	startTag := fmt.Sprintf("<%s", elementName)
	endTag := fmt.Sprintf("</%s>", elementName)
	
	startPos := strings.Index(xmlStr, startTag)
	if startPos == -1 {
		return nil, fmt.Errorf("elemento %s no encontrado", elementName)
	}
	
	// Buscar el cierre del tag inicial
	tagClosePos := strings.Index(xmlStr[startPos:], ">")
	if tagClosePos == -1 {
		return nil, fmt.Errorf("tag de apertura inválido para %s", elementName)
	}
	tagClosePos += startPos + 1
	
	endPos := strings.Index(xmlStr[tagClosePos:], endTag)
	if endPos == -1 {
		return nil, fmt.Errorf("tag de cierre no encontrado para %s", elementName)
	}
	endPos += tagClosePos + len(endTag)
	
	return []byte(xmlStr[startPos:endPos]), nil
}

// ValidateXMLStructure valida que el XML tenga una estructura válida
func (s *UBLService) ValidateXMLStructure(xmlData []byte) error {
	// Verificar que sea XML válido
	var temp interface{}
	if err := xml.Unmarshal(xmlData, &temp); err != nil {
		return fmt.Errorf("XML inválido: %v", err)
	}

	xmlStr := string(xmlData)

	// Verificar que tenga declaración XML
	if !strings.HasPrefix(xmlStr, "<?xml") {
		return fmt.Errorf("declaración XML faltante")
	}

	// Verificar namespaces UBL requeridos
	requiredNamespaces := []string{
		"urn:oasis:names:specification:ubl:schema:xsd",
		"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2",
		"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2",
	}

	for _, ns := range requiredNamespaces {
		if !strings.Contains(xmlStr, ns) {
			return fmt.Errorf("namespace requerido faltante: %s", ns)
		}
	}

	return nil
}

// GetDocumentHash calcula el hash del documento para verificación
func (s *UBLService) GetDocumentHash(xmlData []byte, algorithm string) ([]byte, error) {
	// Esta función será implementada en el servicio de criptografía
	// Por ahora retornamos el XML para que el servicio de firmado lo procese
	return xmlData, nil
}

// CreateUBLExtensions crea la estructura de extensiones UBL vacía
func (s *UBLService) CreateUBLExtensions() *models.UBLExtensions {
	return &models.UBLExtensions{
		UBLExtension: []models.UBLExtension{
			{
				ExtensionContent: &models.ExtensionContent{},
			},
		},
	}
}