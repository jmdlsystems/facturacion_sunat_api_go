package models

import (
	"encoding/xml"
	"time"
)

// UBLInvoice representa una factura en formato UBL 2.1
type UBLInvoice struct {
	XMLName                xml.Name                `xml:"Invoice"`
	Xmlns                  string                  `xml:"xmlns,attr"`
	XmlnsCac               string                  `xml:"xmlns:cac,attr"`
	XmlnsCbc               string                  `xml:"xmlns:cbc,attr"`
	XmlnsDs                string                  `xml:"xmlns:ds,attr"`
	XmlnsExt               string                  `xml:"xmlns:ext,attr"`
	UBLExtensions          *UBLExtensions          `xml:"ext:UBLExtensions"`
	UBLVersionID           string                  `xml:"cbc:UBLVersionID"`
	CustomizationID        string                  `xml:"cbc:CustomizationID"`
	ProfileID              string                  `xml:"cbc:ProfileID,omitempty"`
	ID                     string                  `xml:"cbc:ID"`
	IssueDate              string                  `xml:"cbc:IssueDate"`
	IssueTime              string                  `xml:"cbc:IssueTime,omitempty"`
	DueDate                string                  `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode        string                  `xml:"cbc:InvoiceTypeCode"`
	Note                   []Note                  `xml:"cbc:Note,omitempty"`
	DocumentCurrencyCode   string                  `xml:"cbc:DocumentCurrencyCode"`
	LineCountNumeric       int                     `xml:"cbc:LineCountNumeric"`
	Signature              *Signature              `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty *AccountingSupplierParty `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty *AccountingCustomerParty `xml:"cac:AccountingCustomerParty"`
	PaymentTerms           []PaymentTerms          `xml:"cac:PaymentTerms,omitempty"`
	TaxTotal               []TaxTotal              `xml:"cac:TaxTotal"`
	LegalMonetaryTotal     *LegalMonetaryTotal     `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines           []InvoiceLine           `xml:"cac:InvoiceLine"`
}

// UBLCreditNote representa una nota de crédito en formato UBL 2.1
type UBLCreditNote struct {
	XMLName                xml.Name                `xml:"CreditNote"`
	Xmlns                  string                  `xml:"xmlns,attr"`
	XmlnsCac               string                  `xml:"xmlns:cac,attr"`
	XmlnsCbc               string                  `xml:"xmlns:cbc,attr"`
	XmlnsDs                string                  `xml:"xmlns:ds,attr"`
	XmlnsExt               string                  `xml:"xmlns:ext,attr"`
	UBLExtensions          *UBLExtensions          `xml:"ext:UBLExtensions"`
	UBLVersionID           string                  `xml:"cbc:UBLVersionID"`
	CustomizationID        string                  `xml:"cbc:CustomizationID"`
	ID                     string                  `xml:"cbc:ID"`
	IssueDate              string                  `xml:"cbc:IssueDate"`
	CreditNoteTypeCode     string                  `xml:"cbc:CreditNoteTypeCode"`
	DocumentCurrencyCode   string                  `xml:"cbc:DocumentCurrencyCode"`
	LineCountNumeric       int                     `xml:"cbc:LineCountNumeric"`
	Signature              *Signature              `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty *AccountingSupplierParty `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty *AccountingCustomerParty `xml:"cac:AccountingCustomerParty"`
	TaxTotal               []TaxTotal              `xml:"cac:TaxTotal"`
	LegalMonetaryTotal     *LegalMonetaryTotal     `xml:"cac:LegalMonetaryTotal"`
	CreditNoteLines        []CreditNoteLine        `xml:"cac:CreditNoteLine"`
}

// UBLExtensions para las extensiones UBL
type UBLExtensions struct {
	// SUNAT exige que UBLExtensions esté presente siempre, aunque esté vacío
	UBLExtension []UBLExtension `xml:"ext:UBLExtension"`
}

type UBLExtension struct {
	ExtensionContent *ExtensionContent `xml:"ext:ExtensionContent"`
}

type ExtensionContent struct {
	Signature *DigitalSignature `xml:"ds:Signature,omitempty"`
}

// DigitalSignature representa la firma digital XMLDSig según estándares SUNAT
type DigitalSignature struct {
	XMLName        xml.Name     `xml:"ds:Signature"`
	Id             string       `xml:"Id,attr,omitempty"`
	SignedInfo     *SignedInfo  `xml:"ds:SignedInfo"`
	SignatureValue *SignatureValue `xml:"ds:SignatureValue"`
	KeyInfo        *KeyInfo     `xml:"ds:KeyInfo"`
}

// SignedInfo contiene la información de la firma según XMLDSig
type SignedInfo struct {
	CanonicalizationMethod *CanonicalizationMethod `xml:"ds:CanonicalizationMethod"`
	SignatureMethod        *SignatureMethod        `xml:"ds:SignatureMethod"`
	Reference              *Reference              `xml:"ds:Reference"`
}

// CanonicalizationMethod especifica el algoritmo de canonicalización
type CanonicalizationMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// SignatureMethod especifica el algoritmo de firma
type SignatureMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// Reference especifica la referencia al documento a firmar
type Reference struct {
	URI          string        `xml:"URI,attr"`
	Transforms   *Transforms   `xml:"ds:Transforms"`
	DigestMethod *DigestMethod `xml:"ds:DigestMethod"`
	DigestValue  string        `xml:"ds:DigestValue"`
}

// Transforms especifica las transformaciones a aplicar
type Transforms struct {
	Transform []Transform `xml:"ds:Transform"`
}

// Transform especifica una transformación individual
type Transform struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// DigestMethod especifica el algoritmo de digest
type DigestMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// SignatureValue contiene el valor de la firma en base64
type SignatureValue struct {
	Value string `xml:",chardata"`
}

// KeyInfo contiene información sobre la clave usada para firmar
type KeyInfo struct {
	X509Data *X509Data `xml:"ds:X509Data"`
}

// X509Data contiene el certificado X.509 en base64
type X509Data struct {
	X509Certificate string `xml:"ds:X509Certificate"`
}

// Signature para firma del documento
type Signature struct {
	ID              string          `xml:"cbc:ID"`
	SignatoryParty  *SignatoryParty `xml:"cac:SignatoryParty"`
	DigitalSignatureAttachment *DigitalSignatureAttachment `xml:"cac:DigitalSignatureAttachment"`
}

type SignatoryParty struct {
	PartyIdentification *PartyIdentification `xml:"cac:PartyIdentification"`
	PartyName           *PartyName           `xml:"cac:PartyName"`
}

type DigitalSignatureAttachment struct {
	ExternalReference *ExternalReference `xml:"cac:ExternalReference"`
}

type ExternalReference struct {
	URI string `xml:"cbc:URI"`
}

// AccountingSupplierParty (Emisor)
type AccountingSupplierParty struct {
	Party *Party `xml:"cac:Party"`
}

// AccountingCustomerParty (Receptor)
type AccountingCustomerParty struct {
	Party *Party `xml:"cac:Party"`
}

type Party struct {
	PartyIdentification []PartyIdentification `xml:"cac:PartyIdentification"`
	PartyName           *PartyName            `xml:"cac:PartyName,omitempty"`
	PartyTaxScheme      *PartyTaxScheme       `xml:"cac:PartyTaxScheme,omitempty"`
	PartyLegalEntity    *PartyLegalEntity     `xml:"cac:PartyLegalEntity,omitempty"`
	PostalAddress       *PostalAddress        `xml:"cac:PostalAddress,omitempty"`
	Contact             *Contact              `xml:"cac:Contact,omitempty"`
}

type PartyIdentification struct {
	ID *ID `xml:"cbc:ID"`
}

type ID struct {
	// Para RUC SUNAT: schemeID="6", schemeName="SUNAT:Identificador de Documento de Identidad", schemeAgencyName="PE:SUNAT"
	Value            string `xml:",chardata"`
	SchemeID         string `xml:"schemeID,attr"`           // Nunca omitir
	SchemeName       string `xml:"schemeName,attr"`         // Nunca omitir
	SchemeAgencyName string `xml:"schemeAgencyName,attr"`   // Nunca omitir
	SchemeURI        string `xml:"schemeURI,attr,omitempty"` // Para SUNAT
}

type PartyName struct {
	Name string `xml:"cbc:Name"`
}

type PartyTaxScheme struct {
	RegistrationName string `xml:"cbc:RegistrationName"`
	CompanyID        *ID    `xml:"cbc:CompanyID"`
	TaxScheme        *TaxScheme `xml:"cac:TaxScheme"`
}

type PartyLegalEntity struct {
	RegistrationName      string               `xml:"cbc:RegistrationName"`
	RegistrationAddress   *RegistrationAddress `xml:"cac:RegistrationAddress,omitempty"`
}

type RegistrationAddress struct {
	ID                 *ID                `xml:"cbc:ID,omitempty"`
	AddressTypeCode    string             `xml:"cbc:AddressTypeCode,omitempty"`
	CityName           string             `xml:"cbc:CityName,omitempty"`
	CountrySubentity   string             `xml:"cbc:CountrySubentity,omitempty"`
	District           string             `xml:"cbc:District,omitempty"`
	AddressLine        *AddressLine       `xml:"cac:AddressLine,omitempty"`
	Country            *Country           `xml:"cac:Country,omitempty"`
}

type AddressLine struct {
	Line string `xml:"cbc:Line"`
}

type PostalAddress struct {
	ID                 string `xml:"cbc:ID,omitempty"`
	StreetName         string `xml:"cbc:StreetName,omitempty"`
	CitySubdivisionName string `xml:"cbc:CitySubdivisionName,omitempty"`
	CityName           string `xml:"cbc:CityName,omitempty"`
	CountrySubentity   string `xml:"cbc:CountrySubentity,omitempty"`
	District           string `xml:"cbc:District,omitempty"`
	Country            *Country `xml:"cac:Country,omitempty"`
}

type Country struct {
	IdentificationCode *IdentificationCode `xml:"cbc:IdentificationCode,omitempty"`
}

type IdentificationCode struct {
	ListID           string `xml:"listID,attr,omitempty"`
	ListAgencyName   string `xml:"listAgencyName,attr,omitempty"`
	ListName         string `xml:"listName,attr,omitempty"`
	Value            string `xml:",chardata"`
}

type Contact struct {
	Name            string `xml:"cbc:Name,omitempty"`
	Telephone       string `xml:"cbc:Telephone,omitempty"`
	ElectronicMail  string `xml:"cbc:ElectronicMail,omitempty"`
}

// TaxTotal para impuestos
type TaxTotal struct {
	TaxAmount   *Amount     `xml:"cbc:TaxAmount"`
	TaxSubtotal []TaxSubtotal `xml:"cac:TaxSubtotal"`
}

type TaxSubtotal struct {
	TaxableAmount *Amount   `xml:"cbc:TaxableAmount"`
	TaxAmount     *Amount   `xml:"cbc:TaxAmount"`
	TaxCategory   *TaxCategory `xml:"cac:TaxCategory"`
}

type TaxCategory struct {
	ID         string     `xml:"cbc:ID"`
	Percent    float64    `xml:"cbc:Percent,omitempty"`
	TaxExemptionReasonCode string `xml:"cbc:TaxExemptionReasonCode,omitempty"`
	TaxScheme  *TaxScheme `xml:"cac:TaxScheme"`
}

type TaxScheme struct {
	ID           *ID    `xml:"cbc:ID"`
	Name         string `xml:"cbc:Name"`
	TaxTypeCode  string `xml:"cbc:TaxTypeCode,omitempty"`
}

// LegalMonetaryTotal para totales
type LegalMonetaryTotal struct {
	LineExtensionAmount *Amount `xml:"cbc:LineExtensionAmount,omitempty"`
	TaxExclusiveAmount  *Amount `xml:"cbc:TaxExclusiveAmount,omitempty"`
	TaxInclusiveAmount  *Amount `xml:"cbc:TaxInclusiveAmount,omitempty"`
	ChargeTotalAmount   *Amount `xml:"cbc:ChargeTotalAmount,omitempty"`
	PrepaidAmount       *Amount `xml:"cbc:PrepaidAmount,omitempty"`
	PayableAmount       *Amount `xml:"cbc:PayableAmount"`
}

type Amount struct {
	Value      float64 `xml:",chardata"`
	CurrencyID string  `xml:"currencyID,attr"` // Siempre requerido por SUNAT
}

// InvoiceLine para líneas de factura
type InvoiceLine struct {
	ID                    string                 `xml:"cbc:ID"`
	InvoicedQuantity      *Quantity              `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount   *Amount                `xml:"cbc:LineExtensionAmount"`
	PricingReference      *PricingReference      `xml:"cac:PricingReference,omitempty"`
	TaxTotal              []TaxTotal             `xml:"cac:TaxTotal,omitempty"`
	Item                  *UBLItem               `xml:"cac:Item"`
	Price                 *Price                 `xml:"cac:Price"`
}

// CreditNoteLine para líneas de nota de crédito
type CreditNoteLine struct {
	ID                    string                 `xml:"cbc:ID"`
	CreditedQuantity      *Quantity              `xml:"cbc:CreditedQuantity"`
	LineExtensionAmount   *Amount                `xml:"cbc:LineExtensionAmount"`
	PricingReference      *PricingReference      `xml:"cac:PricingReference,omitempty"`
	TaxTotal              []TaxTotal             `xml:"cac:TaxTotal,omitempty"`
	Item                  *UBLItem               `xml:"cac:Item"`
	Price                 *Price                 `xml:"cac:Price"`
}

// UBLDebitNote representa una nota de débito en formato UBL 2.1
// Estructura basada en los ejemplos oficiales SUNAT
// Referencia: guia+xml+nota de crédito+version 2-1+1+0_0_0 (2).pdf y guia+xml+boleta+version 2-1+1+0_0_0 (2).pdf
// Puedes ajustar los campos según el PDF de nota de débito si tienes diferencias

type UBLDebitNote struct {
	XMLName                xml.Name                `xml:"DebitNote"`
	Xmlns                  string                  `xml:"xmlns,attr"`
	XmlnsCac               string                  `xml:"xmlns:cac,attr"`
	XmlnsCbc               string                  `xml:"xmlns:cbc,attr"`
	XmlnsDs                string                  `xml:"xmlns:ds,attr"`
	XmlnsExt               string                  `xml:"xmlns:ext,attr"`
	UBLExtensions          *UBLExtensions          `xml:"ext:UBLExtensions"`
	UBLVersionID           string                  `xml:"cbc:UBLVersionID"`
	CustomizationID        string                  `xml:"cbc:CustomizationID"`
	ID                     string                  `xml:"cbc:ID"`
	IssueDate              string                  `xml:"cbc:IssueDate"`
	DebitNoteTypeCode      string                  `xml:"cbc:DebitNoteTypeCode"`
	DocumentCurrencyCode   string                  `xml:"cbc:DocumentCurrencyCode"`
	LineCountNumeric       int                     `xml:"cbc:LineCountNumeric"`
	Signature              *Signature              `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty *AccountingSupplierParty `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty *AccountingCustomerParty `xml:"cac:AccountingCustomerParty"`
	TaxTotal               []TaxTotal              `xml:"cac:TaxTotal"`
	LegalMonetaryTotal     *LegalMonetaryTotal     `xml:"cac:LegalMonetaryTotal"`
	DebitNoteLines         []DebitNoteLine         `xml:"cac:DebitNoteLine"`
}

type DebitNoteLine struct {
	ID                    string                 `xml:"cbc:ID"`
	DebitedQuantity       *Quantity              `xml:"cbc:DebitedQuantity"`
	LineExtensionAmount   *Amount                `xml:"cbc:LineExtensionAmount"`
	PricingReference      *PricingReference      `xml:"cac:PricingReference,omitempty"`
	TaxTotal              []TaxTotal             `xml:"cac:TaxTotal,omitempty"`
	Item                  *UBLItem               `xml:"cac:Item"`
	Price                 *Price                 `xml:"cac:Price"`
}

type Quantity struct {
	Value    float64 `xml:",chardata"`
	UnitCode string  `xml:"unitCode,attr"` // Siempre requerido por SUNAT
}

type PricingReference struct {
	AlternativeConditionPrice []AlternativeConditionPrice `xml:"cac:AlternativeConditionPrice"`
}

type AlternativeConditionPrice struct {
	PriceAmount   *Amount `xml:"cbc:PriceAmount"`
	PriceTypeCode string  `xml:"cbc:PriceTypeCode"`
}

type UBLItem struct {
	Description                []string                    `xml:"cbc:Description"`
	SellersItemIdentification  *SellersItemIdentification  `xml:"cac:SellersItemIdentification,omitempty"`
	CommodityClassification    []CommodityClassification   `xml:"cac:CommodityClassification,omitempty"`
	ClassifiedTaxCategory      []ClassifiedTaxCategory     `xml:"cac:ClassifiedTaxCategory,omitempty"`
}

type SellersItemIdentification struct {
	ID string `xml:"cbc:ID"`
}

type CommodityClassification struct {
	ItemClassificationCode *ItemClassificationCode `xml:"cbc:ItemClassificationCode"`
}

type ItemClassificationCode struct {
	Value         string `xml:",chardata"`
	ListID        string `xml:"listID,attr,omitempty"`
	ListAgencyName string `xml:"listAgencyName,attr,omitempty"`
}

type ClassifiedTaxCategory struct {
	ID                        string     `xml:"cbc:ID"`
	Percent                   float64    `xml:"cbc:Percent,omitempty"`
	TaxExemptionReasonCode    string     `xml:"cbc:TaxExemptionReasonCode,omitempty"`
	TierRange                 string     `xml:"cbc:TierRange,omitempty"`
	TaxScheme                 *TaxScheme `xml:"cac:TaxScheme"`
}

type Price struct {
	PriceAmount *Amount `xml:"cbc:PriceAmount"`
}

// Note para monto en letras y otros usos
// Permite atributos como languageLocaleID
type Note struct {
	Value            string `xml:",chardata"`
	LanguageLocaleID string `xml:"languageLocaleID,attr,omitempty"`
}

// PaymentTerms para cuotas de pago y condiciones de pago
type PaymentTerms struct {
	ID             string  `xml:"cbc:ID"`
	PaymentMeansID string  `xml:"cbc:PaymentMeansID,omitempty"`
	Amount         *Amount `xml:"cbc:Amount,omitempty"`
	PaymentDueDate string  `xml:"cbc:PaymentDueDate,omitempty"`
}

// UBLConstants contiene las constantes para UBL 2.1 según guías oficiales SUNAT
type UBLConstants struct {
	Version                    string
	CustomizationID           string
	Xmlns                     string
	XmlnsCac                  string
	XmlnsCbc                  string
	XmlnsDs                   string
	XmlnsExt                  string
	SignatureAlgorithm        string
	DigestAlgorithm           string
	CanonicalizationAlgorithm string
}

var UBLConst = UBLConstants{
	Version:                    "2.1",
	CustomizationID:           "2.0", // SUNAT especifica 2.0 para Perú
	Xmlns:                     "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2",
	XmlnsCac:                  "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2",
	XmlnsCbc:                  "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2",
	XmlnsDs:                   "http://www.w3.org/2000/09/xmldsig#",
	XmlnsExt:                  "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2",
	SignatureAlgorithm:        "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
	DigestAlgorithm:           "http://www.w3.org/2001/04/xmlenc#sha256",
	CanonicalizationAlgorithm: "http://www.w3.org/TR/2001/REC-xml-c14n-20010315",
}

// Constantes específicas para SUNAT
var SUNATConstants = struct {
	// Tipos de documento según catálogo 01
	InvoiceTypeCode        string
	CreditNoteTypeCode     string
	DebitNoteTypeCode      string
	BoletaTypeCode         string
	
	// Códigos de impuestos según catálogo 05
	IGVCode                string
	ICBCode                string
	ISCCode                string
	
	// Monedas según catálogo 02
	PENCurrency            string
	USDCurrency            string
	
	// Unidades de medida según catálogo 03
	NIUUnit                string
	ZZUnit                 string
	
	// Tipos de documento de identidad según catálogo 06
	RUCDocumentType        string
	DNIDocumentType        string
	CEXDocumentType        string
	PASDocumentType        string
}{
	InvoiceTypeCode:    "01",
	CreditNoteTypeCode: "07",
	DebitNoteTypeCode:  "08",
	BoletaTypeCode:     "03",
	
	IGVCode: "1000",
	ICBCode: "2000",
	ISCCode: "9995",
	
	PENCurrency: "PEN",
	USDCurrency: "USD",
	
	NIUUnit: "NIU",
	ZZUnit:  "ZZ",
	
	RUCDocumentType: "6",
	DNIDocumentType: "1",
	CEXDocumentType: "4",
	PASDocumentType: "7",
}

// Helper methods para conversión de tiempo
func FormatUBLDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatUBLDateTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}