package services

import (
	"facturacion_sunat_api_go/internal/models"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ConversionService struct {
	UBLService *UBLService
}

func NewConversionService(ublService *UBLService) *ConversionService {
	return &ConversionService{
		UBLService: ublService,
	}
}

// ConvertToUBL convierte un comprobante de negocio a formato UBL
func (s *ConversionService) ConvertToUBL(comprobante *models.Comprobante) (interface{}, error) {
	// Validación básica de campos obligatorios y catálogos SUNAT
	if comprobante.Emisor.RUC == "" || len(comprobante.Emisor.RUC) != 11 {
		return nil, fmt.Errorf("RUC emisor es obligatorio y debe tener 11 dígitos")
	}
	if comprobante.Receptor.NumeroDocumento == "" {
		return nil, fmt.Errorf("Número de documento del receptor es obligatorio")
	}
	if comprobante.TipoMoneda == "" {
		return nil, fmt.Errorf("Código de moneda es obligatorio")
	}
	if len(comprobante.Items) == 0 {
		return nil, fmt.Errorf("Debe haber al menos un item en el comprobante")
	}
	// Validar tipo de comprobante según catálogo SUNAT
	switch comprobante.Tipo {
	case models.TipoFactura:
		return s.convertToUBLInvoice(comprobante)
	case models.TipoBoleta:
		// Para boleta, usar estructura de factura pero con InvoiceTypeCode "03"
		return s.convertToUBLInvoice(comprobante)
	case models.TipoNotaCredito:
		return s.convertToUBLCreditNote(comprobante)
	case models.TipoNotaDebito:
		return s.convertToUBLRealDebitNote(comprobante)
	default:
		return nil, fmt.Errorf("tipo de comprobante no soportado: %v", comprobante.Tipo)
	}
}

func (s *ConversionService) convertToUBLInvoice(comprobante *models.Comprobante) (*models.UBLInvoice, error) {
	invoice := &models.UBLInvoice{
		Xmlns:                models.UBLConst.Xmlns,
		XmlnsCac:             models.UBLConst.XmlnsCac,
		XmlnsCbc:             models.UBLConst.XmlnsCbc,
		XmlnsDs:              models.UBLConst.XmlnsDs,
		XmlnsExt:             models.UBLConst.XmlnsExt,
		UBLExtensions:        &models.UBLExtensions{
			UBLExtension: []models.UBLExtension{
				{
					ExtensionContent: &models.ExtensionContent{},
				},
			},
		},
		UBLVersionID:         models.UBLConst.Version,
		CustomizationID:      models.UBLConst.CustomizationID,
		ID:                   fmt.Sprintf("%s-%s", comprobante.Serie, comprobante.Numero),
		IssueDate:            models.FormatUBLDate(comprobante.FechaEmision),
		InvoiceTypeCode:      s.getInvoiceTypeCode(comprobante.Tipo),
		DocumentCurrencyCode: comprobante.TipoMoneda,
		LineCountNumeric:     len(comprobante.Items),
	}

	// Fecha de vencimiento (opcional)
	if comprobante.FechaVencimiento != nil {
		invoice.DueDate = models.FormatUBLDate(*comprobante.FechaVencimiento)
	}

	// Mapeo de Note (monto en letras)
	if comprobante.Observaciones != "" {
		invoice.Note = []models.Note{{
			Value: comprobante.Observaciones,
		}}
	}

	// Mapeo de PaymentTerms (cuotas de pago y forma de pago)
	if comprobante.FormaPago != nil {
		var paymentTerms []models.PaymentTerms
		// Si es crédito y hay cuotas
		if strings.ToLower(comprobante.FormaPago.TipoPago) == "credito" && len(comprobante.FormaPago.Cuotas) > 0 {
			// Total de crédito
			paymentTerms = append(paymentTerms, models.PaymentTerms{
				ID: "FormaPago",
				PaymentMeansID: "Credito",
				Amount: &models.Amount{
					Value: comprobante.Totales.ImporteTotal,
					CurrencyID: comprobante.TipoMoneda,
				},
			})
			// Cuotas
			for i, cuota := range comprobante.FormaPago.Cuotas {
				paymentTerms = append(paymentTerms, models.PaymentTerms{
					ID: "FormaPago",
					PaymentMeansID: fmt.Sprintf("Cuota%03d", i+1),
					Amount: &models.Amount{
						Value: cuota.Monto,
						CurrencyID: comprobante.TipoMoneda,
					},
					PaymentDueDate: models.FormatUBLDate(cuota.FechaVencimiento),
				})
			}
		} else {
			// Pago contado o sin cuotas
			paymentTerms = append(paymentTerms, models.PaymentTerms{
				ID: "FormaPago",
				PaymentMeansID: comprobante.FormaPago.TipoPago,
				Amount: &models.Amount{
					Value: comprobante.Totales.ImporteTotal,
					CurrencyID: comprobante.TipoMoneda,
				},
			})
		}
		invoice.PaymentTerms = paymentTerms
	}

	// Proveedor (Emisor)
	supplierParty, err := s.convertSupplierParty(comprobante.Emisor)
	if err != nil {
		return nil, fmt.Errorf("error converting supplier party: %v", err)
	}
	invoice.AccountingSupplierParty = supplierParty

	// Cliente (Receptor)
	customerParty, err := s.convertCustomerParty(comprobante.Receptor)
	if err != nil {
		return nil, fmt.Errorf("error converting customer party: %v", err)
	}
	invoice.AccountingCustomerParty = customerParty

	// Totales de impuestos
	taxTotals, err := s.convertTaxTotals(comprobante.Impuestos, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting tax totals: %v", err)
	}
	invoice.TaxTotal = taxTotals

	// Totales monetarios
	monetaryTotal, err := s.convertLegalMonetaryTotal(comprobante.Totales, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting monetary total: %v", err)
	}
	invoice.LegalMonetaryTotal = monetaryTotal

	// Líneas de factura
	invoiceLines, err := s.convertInvoiceLines(comprobante.Items, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting invoice lines: %v", err)
	}
	invoice.InvoiceLines = invoiceLines

	return invoice, nil
}

// getInvoiceTypeCode retorna el código de tipo de documento según catálogo SUNAT 01
func (s *ConversionService) getInvoiceTypeCode(tipo models.TipoComprobante) string {
	switch tipo {
	case models.TipoFactura:
		return models.SUNATConstants.InvoiceTypeCode
	case models.TipoBoleta:
		return models.SUNATConstants.BoletaTypeCode
	default:
		return models.SUNATConstants.InvoiceTypeCode
	}
}

func (s *ConversionService) convertToUBLCreditNote(comprobante *models.Comprobante) (*models.UBLCreditNote, error) {
	creditNote := &models.UBLCreditNote{
		Xmlns:                models.UBLConst.Xmlns,
		XmlnsCac:             models.UBLConst.XmlnsCac,
		XmlnsCbc:             models.UBLConst.XmlnsCbc,
		XmlnsDs:              models.UBLConst.XmlnsDs,
		XmlnsExt:             models.UBLConst.XmlnsExt,
		UBLExtensions:        &models.UBLExtensions{
			UBLExtension: []models.UBLExtension{
				{
					ExtensionContent: &models.ExtensionContent{},
				},
			},
		},
		UBLVersionID:         models.UBLConst.Version,
		CustomizationID:      models.UBLConst.CustomizationID,
		ID:                   fmt.Sprintf("%s-%s", comprobante.Serie, comprobante.Numero),
		IssueDate:            models.FormatUBLDate(comprobante.FechaEmision),
		CreditNoteTypeCode:   models.SUNATConstants.CreditNoteTypeCode,
		DocumentCurrencyCode: comprobante.TipoMoneda,
		LineCountNumeric:     len(comprobante.Items),
	}

	// Proveedor (Emisor)
	supplierParty, err := s.convertSupplierParty(comprobante.Emisor)
	if err != nil {
		return nil, fmt.Errorf("error converting supplier party: %v", err)
	}
	creditNote.AccountingSupplierParty = supplierParty

	// Cliente (Receptor)
	customerParty, err := s.convertCustomerParty(comprobante.Receptor)
	if err != nil {
		return nil, fmt.Errorf("error converting customer party: %v", err)
	}
	creditNote.AccountingCustomerParty = customerParty

	// Totales de impuestos
	taxTotals, err := s.convertTaxTotals(comprobante.Impuestos, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting tax totals: %v", err)
	}
	creditNote.TaxTotal = taxTotals

	// Totales monetarios
	monetaryTotal, err := s.convertLegalMonetaryTotal(comprobante.Totales, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting monetary total: %v", err)
	}
	creditNote.LegalMonetaryTotal = monetaryTotal

	// Líneas de nota de crédito
	creditNoteLines, err := s.convertCreditNoteLines(comprobante.Items, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting credit note lines: %v", err)
	}
	creditNote.CreditNoteLines = creditNoteLines

	return creditNote, nil
}

// Implementación para Nota de Débito UBL siguiendo el estándar SUNAT
func (s *ConversionService) convertToUBLRealDebitNote(comprobante *models.Comprobante) (*models.UBLDebitNote, error) {
	debitNote := &models.UBLDebitNote{
		Xmlns:                models.UBLConst.Xmlns,
		XmlnsCac:             models.UBLConst.XmlnsCac,
		XmlnsCbc:             models.UBLConst.XmlnsCbc,
		XmlnsDs:              models.UBLConst.XmlnsDs,
		XmlnsExt:             models.UBLConst.XmlnsExt,
		UBLExtensions:        &models.UBLExtensions{
			UBLExtension: []models.UBLExtension{
				{
					ExtensionContent: &models.ExtensionContent{},
				},
			},
		},
		UBLVersionID:         models.UBLConst.Version,
		CustomizationID:      models.UBLConst.CustomizationID,
		ID:                   fmt.Sprintf("%s-%s", comprobante.Serie, comprobante.Numero),
		IssueDate:            models.FormatUBLDate(comprobante.FechaEmision),
		DebitNoteTypeCode:    models.SUNATConstants.DebitNoteTypeCode,
		DocumentCurrencyCode: comprobante.TipoMoneda,
		LineCountNumeric:     len(comprobante.Items),
	}

	// Proveedor (Emisor)
	supplierParty, err := s.convertSupplierParty(comprobante.Emisor)
	if err != nil {
		return nil, fmt.Errorf("error converting supplier party: %v", err)
	}
	debitNote.AccountingSupplierParty = supplierParty

	// Cliente (Receptor)
	customerParty, err := s.convertCustomerParty(comprobante.Receptor)
	if err != nil {
		return nil, fmt.Errorf("error converting customer party: %v", err)
	}
	debitNote.AccountingCustomerParty = customerParty

	// Totales de impuestos
	taxTotals, err := s.convertTaxTotals(comprobante.Impuestos, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting tax totals: %v", err)
	}
	debitNote.TaxTotal = taxTotals

	// Totales monetarios
	monetaryTotal, err := s.convertLegalMonetaryTotal(comprobante.Totales, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting monetary total: %v", err)
	}
	debitNote.LegalMonetaryTotal = monetaryTotal

	// Líneas de nota de débito
	debitNoteLines, err := s.convertDebitNoteLines(comprobante.Items, comprobante.TipoMoneda)
	if err != nil {
		return nil, fmt.Errorf("error converting debit note lines: %v", err)
	}
	debitNote.DebitNoteLines = debitNoteLines

	return debitNote, nil
}

// convertSupplierParty genera la estructura UBL del emisor cumpliendo el estándar SUNAT UBL 2.1
func (s *ConversionService) convertSupplierParty(emisor models.Emisor) (*models.AccountingSupplierParty, error) {
	ruc := strings.TrimSpace(emisor.RUC)
	if len(ruc) != 11 {
		return nil, fmt.Errorf("RUC emisor inválido: debe tener 11 dígitos")
	}
	for _, c := range []rune(ruc) {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("RUC emisor inválido: solo debe contener números")
		}
	}
	if err := models.ValidarRUC(ruc); err != nil {
		return nil, fmt.Errorf("RUC emisor inválido: %v", err)
	}
	if strings.TrimSpace(emisor.RazonSocial) == "" {
		return nil, fmt.Errorf("Razón social del emisor es obligatoria")
	}

	ubigeo := emisor.CodigoPostal
	if ubigeo == "" {
		ubigeo = "150101" // valor por defecto Lima
	}
	addressLine := emisor.Direccion
	if addressLine == "" {
		addressLine = "DIRECCION FISCAL"
	}
	codigoPais := emisor.CodigoPais
	if codigoPais == "" {
		codigoPais = "PE"
	}

	party := &models.Party{
		PartyIdentification: []models.PartyIdentification{
			{
				ID: &models.ID{
					Value:            ruc,
					SchemeID:         "6",
					SchemeName:       "Documento de Identidad",
					SchemeAgencyName: "PE:SUNAT",
					SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
				},
			},
		},
		PartyName: &models.PartyName{
			Name: emisor.RazonSocial,
		},
		PartyTaxScheme: &models.PartyTaxScheme{
			RegistrationName: emisor.RazonSocial,
			CompanyID: &models.ID{
				Value:            ruc,
				SchemeID:         "6",
				SchemeName:       "Documento de Identidad",
				SchemeAgencyName: "PE:SUNAT",
				SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
			},
			TaxScheme: &models.TaxScheme{
				ID: &models.ID{
					Value:            ruc,
					SchemeID:         "6",
					SchemeName:       "Documento de Identidad",
					SchemeAgencyName: "PE:SUNAT",
					SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
				},
			},
		},
		PartyLegalEntity: &models.PartyLegalEntity{
			RegistrationName: emisor.RazonSocial,
			RegistrationAddress: &models.RegistrationAddress{
				ID: &models.ID{
					Value:            ubigeo,
					SchemeID:         "Ubigeo",
					SchemeName:       "Ubigeos",
					SchemeAgencyName: "PE:INEI",
				},
				AddressTypeCode: "0000",
				CityName:        emisor.Provincia,
				CountrySubentity: emisor.Departamento,
				District:        emisor.Distrito,
				AddressLine: &models.AddressLine{
					Line: addressLine,
				},
				Country: &models.Country{
					IdentificationCode: &models.IdentificationCode{
						ListID:           "ISO 3166-1",
						ListAgencyName:   "United Nations Economic Commission for Europe",
						ListName:         "Country",
						Value:            codigoPais,
					},
				},
			},
		},
		PostalAddress: &models.PostalAddress{
			StreetName:         addressLine,
			CitySubdivisionName: emisor.Distrito,
			CityName:           emisor.Provincia,
			CountrySubentity:   emisor.Departamento,
			Country: &models.Country{
				IdentificationCode: &models.IdentificationCode{
					ListID:           "ISO 3166-1",
					ListAgencyName:   "United Nations Economic Commission for Europe",
					ListName:         "Country",
					Value:            codigoPais,
				},
			},
		},
	}

	if emisor.Telefono != "" || emisor.Email != "" {
		party.Contact = &models.Contact{
			Name:            emisor.NombreComercial,
			Telephone:       emisor.Telefono,
			ElectronicMail:  emisor.Email,
		}
	}

	return &models.AccountingSupplierParty{
		Party: party,
	}, nil
}

func (s *ConversionService) convertCustomerParty(receptor models.Receptor) (*models.AccountingCustomerParty, error) {
	schemeID := "6" // Por defecto RUC
	if receptor.TipoDocumento == "1" {
		schemeID = "1" // DNI
	} else if receptor.TipoDocumento == "4" {
		schemeID = "4" // CE
	} else if receptor.TipoDocumento == "7" {
		schemeID = "7" // Pasaporte
	}
	if strings.TrimSpace(receptor.RazonSocial) == "" {
		return nil, fmt.Errorf("Razón social del receptor es obligatoria")
	}
	if receptor.NumeroDocumento == "" {
		return nil, fmt.Errorf("Número de documento del receptor es obligatorio")
	}
	addressLine := receptor.Direccion
	if addressLine == "" {
		addressLine = "DIRECCION CLIENTE"
	}

	party := &models.Party{
		PartyIdentification: []models.PartyIdentification{
			{
				ID: &models.ID{
					Value:            receptor.NumeroDocumento,
					SchemeID:         schemeID,
					SchemeName:       "Documento de Identidad",
					SchemeAgencyName: "PE:SUNAT",
					SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
				},
			},
		},
		PartyName: &models.PartyName{
			Name: receptor.RazonSocial,
		},
		PartyTaxScheme: &models.PartyTaxScheme{
			RegistrationName: receptor.RazonSocial,
			CompanyID: &models.ID{
				Value:            receptor.NumeroDocumento,
				SchemeID:         schemeID,
				SchemeName:       "Documento de Identidad",
				SchemeAgencyName: "PE:SUNAT",
				SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
			},
			TaxScheme: &models.TaxScheme{
				ID: &models.ID{
					Value:            receptor.NumeroDocumento,
					SchemeID:         schemeID,
					SchemeName:       "Documento de Identidad",
					SchemeAgencyName: "PE:SUNAT",
					SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo06",
				},
			},
		},
		PartyLegalEntity: &models.PartyLegalEntity{
			RegistrationName: receptor.RazonSocial,
			RegistrationAddress: &models.RegistrationAddress{
				CityName:         "LIMA",
				CountrySubentity: "LIMA",
				District:         "LIMA",
				AddressLine: &models.AddressLine{
					Line: addressLine,
				},
				Country: &models.Country{
					IdentificationCode: &models.IdentificationCode{
						ListID:           "ISO 3166-1",
						ListAgencyName:   "United Nations Economic Commission for Europe",
						ListName:         "Country",
						Value:            "PE",
					},
				},
			},
		},
	}

	if receptor.Direccion != "" {
		party.PostalAddress = &models.PostalAddress{
			StreetName: receptor.Direccion,
			Country: &models.Country{
				IdentificationCode: &models.IdentificationCode{
					ListID:           "ISO 3166-1",
					ListAgencyName:   "United Nations Economic Commission for Europe",
					ListName:         "Country",
					Value:            "PE",
				},
			},
		}
	}

	if receptor.Email != "" {
		party.Contact = &models.Contact{
			ElectronicMail: receptor.Email,
		}
	}

	return &models.AccountingCustomerParty{
		Party: party,
	}, nil
}

func (s *ConversionService) convertTaxTotals(impuestos []models.Impuesto, moneda string) ([]models.TaxTotal, error) {
	var taxTotals []models.TaxTotal

	// Agrupar impuestos por tipo
	taxGroups := make(map[string][]models.Impuesto)
	for _, impuesto := range impuestos {
		key := impuesto.TipoImpuesto + "-" + impuesto.CodigoImpuesto
		taxGroups[key] = append(taxGroups[key], impuesto)
	}

	for _, group := range taxGroups {
		var totalAmount float64
		var taxSubtotals []models.TaxSubtotal

		for _, impuesto := range group {
			totalAmount += impuesto.MontoImpuesto

			taxSubtotal := models.TaxSubtotal{
				TaxableAmount: &models.Amount{
					Value:      impuesto.BaseImponible,
					CurrencyID: moneda,
				},
				TaxAmount: &models.Amount{
					Value:      impuesto.MontoImpuesto,
					CurrencyID: moneda,
				},
				TaxCategory: &models.TaxCategory{
					ID:      impuesto.CodigoImpuesto,
					Percent: impuesto.Tasa,
					TaxScheme: &models.TaxScheme{
						ID: &models.ID{
							Value:            impuesto.TipoImpuesto,
							SchemeID:         "UN/ECE 5153",
							SchemeName:       "Codigo de tributos",
							SchemeAgencyName: "PE:SUNAT",
						},
						Name:        s.getTaxSchemeName(impuesto.TipoImpuesto),
						TaxTypeCode: "VAT",
					},
				},
			}

			taxSubtotals = append(taxSubtotals, taxSubtotal)
		}

		taxTotal := models.TaxTotal{
			TaxAmount: &models.Amount{
				Value:      totalAmount,
				CurrencyID: moneda,
			},
			TaxSubtotal: taxSubtotals,
		}

		taxTotals = append(taxTotals, taxTotal)
	}

	return taxTotals, nil
}

func (s *ConversionService) convertLegalMonetaryTotal(totales models.Totales, moneda string) (*models.LegalMonetaryTotal, error) {
	return &models.LegalMonetaryTotal{
		LineExtensionAmount: &models.Amount{
			Value:      totales.TotalValorVenta,
			CurrencyID: moneda,
		},
		TaxExclusiveAmount: &models.Amount{
			Value:      totales.TotalValorVenta,
			CurrencyID: moneda,
		},
		TaxInclusiveAmount: &models.Amount{
			Value:      totales.TotalPrecioVenta,
			CurrencyID: moneda,
		},
		ChargeTotalAmount: &models.Amount{
			Value:      totales.TotalDescuentos,
			CurrencyID: moneda,
		},
		PrepaidAmount: &models.Amount{
			Value:      totales.TotalAnticipos,
			CurrencyID: moneda,
		},
		PayableAmount: &models.Amount{
			Value:      totales.ImporteTotal,
			CurrencyID: moneda,
		},
	}, nil
}

func (s *ConversionService) convertInvoiceLines(items []models.Item, moneda string) ([]models.InvoiceLine, error) {
	var invoiceLines []models.InvoiceLine

	for _, item := range items {
		invoiceLine := models.InvoiceLine{
			ID: strconv.Itoa(item.ID),
			InvoicedQuantity: &models.Quantity{
				Value:    item.Cantidad,
				UnitCode: item.UnidadMedida,
			},
			LineExtensionAmount: &models.Amount{
				Value:      item.ValorVenta,
				CurrencyID: moneda,
			},
			Item: &models.UBLItem{
				Description: []string{item.Descripcion},
				SellersItemIdentification: &models.SellersItemIdentification{
					ID: item.Codigo,
				},
				ClassifiedTaxCategory: s.convertClassifiedTaxCategory(item),
			},
			Price: &models.Price{
				PriceAmount: &models.Amount{
					Value:      item.PrecioUnitario,
					CurrencyID: moneda,
				},
			},
		}

		// Referencia de precios (para diferentes tipos de afectación)
		if item.TipoAfectacion != models.GravadoOneroso {
			invoiceLine.PricingReference = &models.PricingReference{
				AlternativeConditionPrice: []models.AlternativeConditionPrice{
					{
						PriceAmount: &models.Amount{
							Value:      item.ValorUnitario,
							CurrencyID: moneda,
						},
						PriceTypeCode: "01", // Precio unitario (incluye el IGV)
					},
				},
			}
		}

		// Impuestos por línea
		if len(item.ImpuestoItem) > 0 {
			taxTotals, err := s.convertItemTaxTotals(item.ImpuestoItem, moneda)
			if err != nil {
				return nil, fmt.Errorf("error converting item tax totals: %v", err)
			}
			invoiceLine.TaxTotal = taxTotals
		}

		invoiceLines = append(invoiceLines, invoiceLine)
	}

	return invoiceLines, nil
}

func (s *ConversionService) convertCreditNoteLines(items []models.Item, moneda string) ([]models.CreditNoteLine, error) {
	var creditNoteLines []models.CreditNoteLine

	for _, item := range items {
		creditNoteLine := models.CreditNoteLine{
			ID: strconv.Itoa(item.ID),
			CreditedQuantity: &models.Quantity{
				Value:    item.Cantidad,
				UnitCode: item.UnidadMedida,
			},
			LineExtensionAmount: &models.Amount{
				Value:      item.ValorVenta,
				CurrencyID: moneda,
			},
			Item: &models.UBLItem{
				Description: []string{item.Descripcion},
				SellersItemIdentification: &models.SellersItemIdentification{
					ID: item.Codigo,
				},
				ClassifiedTaxCategory: s.convertClassifiedTaxCategory(item),
			},
			Price: &models.Price{
				PriceAmount: &models.Amount{
					Value:      item.PrecioUnitario,
					CurrencyID: moneda,
				},
			},
		}

		creditNoteLines = append(creditNoteLines, creditNoteLine)
	}

	return creditNoteLines, nil
}

// Conversión de items a DebitNoteLine
func (s *ConversionService) convertDebitNoteLines(items []models.Item, moneda string) ([]models.DebitNoteLine, error) {
	var debitNoteLines []models.DebitNoteLine

	for _, item := range items {
		debitNoteLine := models.DebitNoteLine{
			ID: strconv.Itoa(item.ID),
			DebitedQuantity: &models.Quantity{
				Value:    item.Cantidad,
				UnitCode: item.UnidadMedida,
			},
			LineExtensionAmount: &models.Amount{
				Value:      item.ValorVenta,
				CurrencyID: moneda,
			},
			Item: &models.UBLItem{
				Description: []string{item.Descripcion},
				SellersItemIdentification: &models.SellersItemIdentification{
					ID: item.Codigo,
				},
				ClassifiedTaxCategory: s.convertClassifiedTaxCategory(item),
			},
			Price: &models.Price{
				PriceAmount: &models.Amount{
					Value:      item.ValorUnitario,
					CurrencyID: moneda,
				},
			},
		}

		// Referencia de precios (para diferentes tipos de afectación)
		if item.TipoAfectacion != models.GravadoOneroso {
			debitNoteLine.PricingReference = &models.PricingReference{
				AlternativeConditionPrice: []models.AlternativeConditionPrice{
					{
						PriceAmount: &models.Amount{
							Value:      item.ValorUnitario,
							CurrencyID: moneda,
						},
						PriceTypeCode: "01", // Precio unitario (incluye el IGV)
					},
				},
			}
		}

		// Impuestos por línea
		if len(item.ImpuestoItem) > 0 {
			taxTotals, err := s.convertItemTaxTotals(item.ImpuestoItem, moneda)
			if err != nil {
				return nil, fmt.Errorf("error converting item tax totals: %v", err)
			}
			debitNoteLine.TaxTotal = taxTotals
		}

		debitNoteLines = append(debitNoteLines, debitNoteLine)
	}

	return debitNoteLines, nil
}

// Cambia el ID de TaxCategory para operaciones gravadas a "S"
func (s *ConversionService) convertClassifiedTaxCategory(item models.Item) []models.ClassifiedTaxCategory {
	var categories []models.ClassifiedTaxCategory

	var id, taxExemptionReasonCode string
	percent := 18.0 // Por defecto IGV, puedes parametrizar si es variable

	switch item.TipoAfectacion.String() {
	case "10": // Gravado - Operación Onerosa
		id = "S"
		taxExemptionReasonCode = "10"
	case "20": // Exonerado
		id = "E"
		taxExemptionReasonCode = "20"
	case "30": // Inafecto
		id = "O"
		taxExemptionReasonCode = "30"
	case "11": // Gravado - Operación Gratuita (Bonificación)
		id = "G"
		taxExemptionReasonCode = "11"
	case "21": // Transferencia gratuita
		id = "G"
		taxExemptionReasonCode = "21"
	// Puedes agregar más casos según catálogo 07
	default:
		id = "S"
		taxExemptionReasonCode = "10"
	}

	categories = append(categories, models.ClassifiedTaxCategory{
		ID:                        id,
		Percent:                   percent,
		TaxExemptionReasonCode:    taxExemptionReasonCode,
		TaxScheme: &models.TaxScheme{
			ID: &models.ID{
				Value:            "1000",
				SchemeID:         "UN/ECE 5153",
				SchemeName:       "Codigo de tributos",
				SchemeAgencyName: "PE:SUNAT",
			},
			Name:        "IGV",
			TaxTypeCode: "VAT",
		},
	})
	return categories
}

func (s *ConversionService) convertItemTaxTotals(impuestosItem []models.ImpuestoItem, moneda string) ([]models.TaxTotal, error) {
	var taxTotals []models.TaxTotal

	for _, impuesto := range impuestosItem {
		taxTotal := models.TaxTotal{
			TaxAmount: &models.Amount{
				Value:      impuesto.MontoImpuesto,
				CurrencyID: moneda,
			},
			TaxSubtotal: []models.TaxSubtotal{
				{
					TaxableAmount: &models.Amount{
						Value:      impuesto.BaseImponible,
						CurrencyID: moneda,
					},
					TaxAmount: &models.Amount{
						Value:      impuesto.MontoImpuesto,
						CurrencyID: moneda,
					},
					TaxCategory: &models.TaxCategory{
						ID:      impuesto.CodigoImpuesto,
						Percent: impuesto.Tasa,
						TaxScheme: &models.TaxScheme{
							ID: &models.ID{
								Value:            impuesto.TipoImpuesto,
								SchemeID:         "UN/ECE 5153",
								SchemeName:       "Codigo de tributos",
								SchemeAgencyName: "PE:SUNAT",
							},
							Name: s.getTaxSchemeName(impuesto.TipoImpuesto),
							TaxTypeCode: "VAT",
						},
					},
				},
			},
		}

		taxTotals = append(taxTotals, taxTotal)
	}

	return taxTotals, nil
}

func (s *ConversionService) getTaxSchemeName(tipoImpuesto string) string {
	switch tipoImpuesto {
	case models.SUNATConstants.IGVCode: // IGV
		return "IGV"
	case "1016": // IVAP
		return "IVAP"
	case models.SUNATConstants.ICBCode: // ISC
		return "ISC"
	case models.SUNATConstants.ISCCode: // EXP
		return "EXP"
	case "9996": // GRA
		return "GRA"
	case "9997": // EXO
		return "EXO"
	case "9998": // INA
		return "INA"
	case "9999": // OTROS
		return "OTROS"
	default:
		return "IGV"
	}
}

// CalculateTotals calcula automáticamente los totales del comprobante
func (s *ConversionService) CalculateTotals(comprobante *models.Comprobante) error {
	var totales models.Totales
	var impuestos []models.Impuesto

	// Mapa para agrupar impuestos
	impuestosMap := make(map[string]*models.Impuesto)

	for _, item := range comprobante.Items {
		// Calcular valores del item
		valorVenta := item.Cantidad * item.ValorUnitario
		valorTotal := valorVenta + item.DescuentoUnitario

		// Actualizar item con valores calculados
		item.ValorVenta = valorVenta
		item.ValorTotal = valorTotal

		// Clasificar según tipo de afectación
		switch item.TipoAfectacion {
		case models.GravadoOneroso:
			totales.TotalVentaGravada += valorVenta
		case models.GravadoGratuito:
			totales.TotalVentaGratuita += valorVenta
		case models.Exonerado:
			totales.TotalVentaExonerada += valorVenta
		case models.Inafecto:
			totales.TotalVentaInafecta += valorVenta
		}

		// Procesar impuestos del item
		for _, impuestoItem := range item.ImpuestoItem {
			key := impuestoItem.TipoImpuesto + "-" + impuestoItem.CodigoImpuesto

			if impuesto, exists := impuestosMap[key]; exists {
				impuesto.BaseImponible += impuestoItem.BaseImponible
				impuesto.MontoImpuesto += impuestoItem.MontoImpuesto
			} else {
				impuestosMap[key] = &models.Impuesto{
					TipoImpuesto:   impuestoItem.TipoImpuesto,
					CodigoImpuesto: impuestoItem.CodigoImpuesto,
					BaseImponible:  impuestoItem.BaseImponible,
					Tasa:           impuestoItem.Tasa,
					MontoImpuesto:  impuestoItem.MontoImpuesto,
				}
			}
		}
	}

	// Convertir mapa a slice
	for _, impuesto := range impuestosMap {
		impuestos = append(impuestos, *impuesto)
		totales.TotalImpuestos += impuesto.MontoImpuesto
	}

	// Calcular totales finales
	totales.TotalValorVenta = totales.TotalVentaGravada + totales.TotalVentaExonerada + totales.TotalVentaInafecta
	totales.TotalPrecioVenta = totales.TotalValorVenta + totales.TotalImpuestos
	totales.ImporteTotal = totales.TotalPrecioVenta - totales.TotalDescuentos - totales.TotalAnticipos + totales.Redondeo

	// Redondear a 2 decimales
	totales.ImporteTotal = math.Round(totales.ImporteTotal*100) / 100

	// Actualizar comprobante
	comprobante.Totales = totales
	comprobante.Impuestos = impuestos

	return nil
}