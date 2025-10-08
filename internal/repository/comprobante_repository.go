package repository

import (
	"database/sql"
	"encoding/json"
	"facturacion_sunat_api_go/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type ComprobanteRepository struct {
	db *sql.DB
}

func NewComprobanteRepository(db *sql.DB) *ComprobanteRepository {
	return &ComprobanteRepository{
		db: db,
	}
}

// Create inserta un nuevo comprobante
func (r *ComprobanteRepository) Create(comprobante *models.Comprobante) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %v", err)
	}
	defer tx.Rollback()

	// Generar UUID si no existe
	if comprobante.ID == "" {
		comprobante.ID = uuid.New().String()
	}

	// Insertar comprobante principal
	query := `
		INSERT INTO comprobantes (
			id, tipo, serie, numero, fecha_emision, fecha_vencimiento, tipo_moneda,
			emisor_ruc, emisor_razon_social, emisor_nombre_comercial, emisor_direccion,
			emisor_distrito, emisor_provincia, emisor_departamento, emisor_pais,
			emisor_telefono, emisor_email,
			receptor_tipo_documento, receptor_numero_documento, receptor_razon_social,
			receptor_direccion, receptor_email,
			total_valor_venta, total_impuestos, total_precio_venta, importe_total,
			estado_proceso, observaciones, fecha_creacion, fecha_actualizacion, usuario_creacion
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
		)`

	_, err = tx.Exec(query,
		comprobante.ID, comprobante.Tipo, comprobante.Serie, comprobante.Numero,
		comprobante.FechaEmision, comprobante.FechaVencimiento, comprobante.TipoMoneda,
		comprobante.Emisor.RUC, comprobante.Emisor.RazonSocial, comprobante.Emisor.NombreComercial,
		comprobante.Emisor.Direccion, comprobante.Emisor.Distrito, comprobante.Emisor.Provincia,
		comprobante.Emisor.Departamento, comprobante.Emisor.CodigoPais, comprobante.Emisor.Telefono,
		comprobante.Emisor.Email, comprobante.Receptor.TipoDocumento, comprobante.Receptor.NumeroDocumento,
		comprobante.Receptor.RazonSocial, comprobante.Receptor.Direccion, comprobante.Receptor.Email,
		comprobante.Totales.TotalValorVenta, comprobante.Totales.TotalImpuestos,
		comprobante.Totales.TotalPrecioVenta, comprobante.Totales.ImporteTotal,
		comprobante.EstadoProceso, comprobante.Observaciones, comprobante.FechaCreacion,
		comprobante.FechaActualizacion, comprobante.UsuarioCreacion,
	)
	if err != nil {
		return fmt.Errorf("error insertando comprobante: %v", err)
	}

	// Insertar items
	for _, item := range comprobante.Items {
		if err := r.insertItem(tx, comprobante.ID, item); err != nil {
			return fmt.Errorf("error insertando item: %v", err)
		}
	}

	// Insertar impuestos
	for _, impuesto := range comprobante.Impuestos {
		if err := r.insertImpuesto(tx, comprobante.ID, nil, impuesto); err != nil {
			return fmt.Errorf("error insertando impuesto: %v", err)
		}
	}

	// Insertar totales
	if err := r.insertTotales(tx, comprobante.ID, comprobante.Totales); err != nil {
		return fmt.Errorf("error insertando totales: %v", err)
	}

	return tx.Commit()
}

// GetByID obtiene un comprobante por ID
func (r *ComprobanteRepository) GetByID(id string) (*models.Comprobante, error) {
	query := `
		SELECT id, tipo, serie, numero, fecha_emision, fecha_vencimiento, tipo_moneda,
			emisor_ruc, emisor_razon_social, emisor_nombre_comercial, emisor_direccion,
			emisor_distrito, emisor_provincia, emisor_departamento, emisor_pais,
			emisor_telefono, emisor_email,
			receptor_tipo_documento, receptor_numero_documento, receptor_razon_social,
			receptor_direccion, receptor_email,
			total_valor_venta, total_impuestos, total_precio_venta, importe_total,
			estado_proceso, xml_generado, xml_firmado, ticket_sunat, estado_sunat,
			observaciones, fecha_creacion, fecha_actualizacion, usuario_creacion
		FROM comprobantes WHERE id = $1`

	var comprobante models.Comprobante
	var xmlGenerado, xmlFirmado, ticketSunat, estadoSunat, usuarioCreacion sql.NullString
	var fechaVencimiento sql.NullTime
	var nombreComercial, telefono, email, direccionReceptor, emailReceptor sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&comprobante.ID, &comprobante.Tipo, &comprobante.Serie, &comprobante.Numero,
		&comprobante.FechaEmision, &fechaVencimiento, &comprobante.TipoMoneda,
		&comprobante.Emisor.RUC, &comprobante.Emisor.RazonSocial, &nombreComercial,
		&comprobante.Emisor.Direccion, &comprobante.Emisor.Distrito, &comprobante.Emisor.Provincia,
		&comprobante.Emisor.Departamento, &comprobante.Emisor.CodigoPais, &telefono,
		&email, &comprobante.Receptor.TipoDocumento, &comprobante.Receptor.NumeroDocumento,
		&comprobante.Receptor.RazonSocial, &direccionReceptor, &emailReceptor,
		&comprobante.Totales.TotalValorVenta, &comprobante.Totales.TotalImpuestos,
		&comprobante.Totales.TotalPrecioVenta, &comprobante.Totales.ImporteTotal,
		&comprobante.EstadoProceso, &xmlGenerado, &xmlFirmado, &ticketSunat, &estadoSunat,
		&comprobante.Observaciones, &comprobante.FechaCreacion, &comprobante.FechaActualizacion,
		&usuarioCreacion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("comprobante no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo comprobante: %v", err)
	}

	// Manejar campos nullable
	if fechaVencimiento.Valid {
		comprobante.FechaVencimiento = &fechaVencimiento.Time
	}
	if nombreComercial.Valid {
		comprobante.Emisor.NombreComercial = nombreComercial.String
	}
	if telefono.Valid {
		comprobante.Emisor.Telefono = telefono.String
	}
	if email.Valid {
		comprobante.Emisor.Email = email.String
	}
	if direccionReceptor.Valid {
		comprobante.Receptor.Direccion = direccionReceptor.String
	}
	if emailReceptor.Valid {
		comprobante.Receptor.Email = emailReceptor.String
	}
	if xmlGenerado.Valid {
		comprobante.XMLGenerado = xmlGenerado.String
	}
	if xmlFirmado.Valid {
		comprobante.XMLFirmado = xmlFirmado.String
	}
	if ticketSunat.Valid {
		comprobante.TicketSUNAT = ticketSunat.String
	}
	if estadoSunat.Valid {
		comprobante.EstadoSUNAT = estadoSunat.String
	}
	if usuarioCreacion.Valid {
		comprobante.UsuarioCreacion = usuarioCreacion.String
	}

	// Cargar items
	items, err := r.getItems(comprobante.ID)
	if err != nil {
		return nil, fmt.Errorf("error cargando items: %v", err)
	}
	comprobante.Items = items

	// Cargar impuestos
	impuestos, err := r.getImpuestos(comprobante.ID)
	if err != nil {
		return nil, fmt.Errorf("error cargando impuestos: %v", err)
	}
	comprobante.Impuestos = impuestos

	// Cargar totales completos
	totales, err := r.getTotales(comprobante.ID)
	if err != nil {
		return nil, fmt.Errorf("error cargando totales: %v", err)
	}
	comprobante.Totales = totales

	return &comprobante, nil
}

// List obtiene lista paginada de comprobantes
func (r *ComprobanteRepository) List(page, limit int, filters map[string]interface{}) ([]*models.Comprobante, int, error) {
	offset := (page - 1) * limit

	// Construir WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if estado, ok := filters["estado_proceso"]; ok {
		whereClause += fmt.Sprintf(" AND estado_proceso = $%d", argIndex)
		args = append(args, estado)
		argIndex++
	}

	if tipo, ok := filters["tipo"]; ok {
		whereClause += fmt.Sprintf(" AND tipo = $%d", argIndex)
		args = append(args, tipo)
		argIndex++
	}

	if ruc, ok := filters["emisor_ruc"]; ok {
		whereClause += fmt.Sprintf(" AND emisor_ruc = $%d", argIndex)
		args = append(args, ruc)
		argIndex++
	}

	if fechaDesde, ok := filters["fecha_desde"]; ok {
		whereClause += fmt.Sprintf(" AND fecha_emision >= $%d", argIndex)
		args = append(args, fechaDesde)
		argIndex++
	}

	if fechaHasta, ok := filters["fecha_hasta"]; ok {
		whereClause += fmt.Sprintf(" AND fecha_emision <= $%d", argIndex)
		args = append(args, fechaHasta)
		argIndex++
	}

	// Contar total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM comprobantes %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error contando comprobantes: %v", err)
	}

	// Obtener datos
	query := fmt.Sprintf(`
		SELECT id, tipo, serie, numero, fecha_emision, fecha_vencimiento,
			emisor_ruc, emisor_razon_social, receptor_razon_social,
			importe_total, estado_proceso, fecha_creacion
		FROM comprobantes %s
		ORDER BY fecha_creacion DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error consultando comprobantes: %v", err)
	}
	defer rows.Close()

	var comprobantes []*models.Comprobante
	for rows.Next() {
		comprobante := &models.Comprobante{}
		var fechaVencimiento sql.NullTime

		err := rows.Scan(
			&comprobante.ID, &comprobante.Tipo, &comprobante.Serie, &comprobante.Numero,
			&comprobante.FechaEmision, &fechaVencimiento,
			&comprobante.Emisor.RUC, &comprobante.Emisor.RazonSocial, &comprobante.Receptor.RazonSocial,
			&comprobante.Totales.ImporteTotal, &comprobante.EstadoProceso, &comprobante.FechaCreacion,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error escaneando comprobante: %v", err)
		}

		if fechaVencimiento.Valid {
			comprobante.FechaVencimiento = &fechaVencimiento.Time
		}

		comprobantes = append(comprobantes, comprobante)
	}

	return comprobantes, total, nil
}

// Update actualiza un comprobante
func (r *ComprobanteRepository) Update(comprobante *models.Comprobante) error {
	comprobante.FechaActualizacion = time.Now()

	query := `
		UPDATE comprobantes SET
			fecha_vencimiento = $2, tipo_moneda = $3,
			emisor_razon_social = $4, emisor_direccion = $5,
			receptor_razon_social = $6, receptor_direccion = $7,
			total_valor_venta = $8, total_impuestos = $9,
			total_precio_venta = $10, importe_total = $11,
			observaciones = $12, fecha_actualizacion = $13
		WHERE id = $1`

	_, err := r.db.Exec(query,
		comprobante.ID, comprobante.FechaVencimiento, comprobante.TipoMoneda,
		comprobante.Emisor.RazonSocial, comprobante.Emisor.Direccion,
		comprobante.Receptor.RazonSocial, comprobante.Receptor.Direccion,
		comprobante.Totales.TotalValorVenta, comprobante.Totales.TotalImpuestos,
		comprobante.Totales.TotalPrecioVenta, comprobante.Totales.ImporteTotal,
		comprobante.Observaciones, comprobante.FechaActualizacion,
	)

	if err != nil {
		return fmt.Errorf("error actualizando comprobante: %v", err)
	}

	return nil
}

// Delete elimina un comprobante
func (r *ComprobanteRepository) Delete(id string) error {
	query := "DELETE FROM comprobantes WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error eliminando comprobante: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verificando eliminación: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comprobante no encontrado")
	}

	return nil
}

// UpdateStatus actualiza el estado de un comprobante
func (r *ComprobanteRepository) UpdateStatus(id string, status models.EstadoProceso) error {
	query := `
		UPDATE comprobantes 
		SET estado_proceso = $2, fecha_actualizacion = $3 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando estado: %v", err)
	}

	return nil
}

// UpdateXML actualiza el XML generado
func (r *ComprobanteRepository) UpdateXML(id, xml string) error {
	query := `
		UPDATE comprobantes 
		SET xml_generado = $2, fecha_actualizacion = $3 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, xml, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando XML: %v", err)
	}

	return nil
}

// UpdateSignedXML actualiza el XML firmado
func (r *ComprobanteRepository) UpdateSignedXML(id, xml string) error {
	query := `
		UPDATE comprobantes 
		SET xml_firmado = $2, fecha_actualizacion = $3 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, xml, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando XML firmado: %v", err)
	}

	return nil
}

// UpdateSUNATInfo actualiza información de SUNAT
func (r *ComprobanteRepository) UpdateSUNATInfo(id, ticket, estado string) error {
	query := `
		UPDATE comprobantes 
		SET ticket_sunat = $2, estado_sunat = $3, fecha_actualizacion = $4 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, ticket, estado, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando información SUNAT: %v", err)
	}

	return nil
}

// UpdateCDR actualiza el CDR de SUNAT
func (r *ComprobanteRepository) UpdateCDR(id string, cdr []byte) error {
	query := `
		UPDATE comprobantes 
		SET cdr_sunat = $2, fecha_actualizacion = $3 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, cdr, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando CDR: %v", err)
	}

	return nil
}

// UpdateArchivoZIP actualiza el archivo ZIP del comprobante
func (r *ComprobanteRepository) UpdateArchivoZIP(id string, zip []byte) error {
	query := `
		UPDATE comprobantes 
		SET archivo_zip = $2, fecha_actualizacion = $3 
		WHERE id = $1`

	_, err := r.db.Exec(query, id, zip, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando archivo ZIP: %v", err)
	}

	return nil
}

// Métodos para lotes
type Lote struct {
	ID                   string                 `json:"id"`
	Descripcion          string                 `json:"descripcion"`
	TotalDocumentos      int                    `json:"total_documentos"`
	DocumentosProcesados int                    `json:"documentos_procesados"`
	DocumentosExitosos   int                    `json:"documentos_exitosos"`
	DocumentosFallidos   int                    `json:"documentos_fallidos"`
	Estado               string                 `json:"estado"`
	FechaInicio          *time.Time             `json:"fecha_inicio"`
	FechaFin             *time.Time             `json:"fecha_fin"`
	ConfiguracionProceso map[string]interface{} `json:"configuracion_proceso"`
	FechaCreacion        time.Time              `json:"fecha_creacion"`
	FechaActualizacion   time.Time              `json:"fecha_actualizacion"`
	UsuarioCreacion      string                 `json:"usuario_creacion"`
}

// CreateBatch crea un nuevo lote
func (r *ComprobanteRepository) CreateBatch(batch *Lote) error {
	if batch.ID == "" {
		batch.ID = uuid.New().String()
	}

	configJSON, _ := json.Marshal(batch.ConfiguracionProceso)

	query := `
		INSERT INTO lotes (
			id, descripcion, total_documentos, estado, fecha_inicio,
			configuracion_proceso, fecha_creacion, usuario_creacion
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(query,
		batch.ID, batch.Descripcion, batch.TotalDocumentos, batch.Estado,
		batch.FechaInicio, configJSON, time.Now(), batch.UsuarioCreacion,
	)

	if err != nil {
		return fmt.Errorf("error creando lote: %v", err)
	}

	return nil
}

// GetBatch obtiene un lote por ID
func (r *ComprobanteRepository) GetBatch(id string) (*Lote, error) {
	query := `
		SELECT id, descripcion, total_documentos, documentos_procesados,
			documentos_exitosos, documentos_fallidos, estado, fecha_inicio,
			fecha_fin, configuracion_proceso, fecha_creacion, fecha_actualizacion,
			usuario_creacion
		FROM lotes WHERE id = $1`

	var lote Lote
	var fechaInicio, fechaFin sql.NullTime
	var configJSON []byte
	var usuarioCreacion sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&lote.ID, &lote.Descripcion, &lote.TotalDocumentos, &lote.DocumentosProcesados,
		&lote.DocumentosExitosos, &lote.DocumentosFallidos, &lote.Estado,
		&fechaInicio, &fechaFin, &configJSON, &lote.FechaCreacion,
		&lote.FechaActualizacion, &usuarioCreacion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lote no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo lote: %v", err)
	}

	if fechaInicio.Valid {
		lote.FechaInicio = &fechaInicio.Time
	}
	if fechaFin.Valid {
		lote.FechaFin = &fechaFin.Time
	}
	if usuarioCreacion.Valid {
		lote.UsuarioCreacion = usuarioCreacion.String
	}
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &lote.ConfiguracionProceso)
	}

	return &lote, nil
}

// UpdateBatchProgress actualiza el progreso de un lote
func (r *ComprobanteRepository) UpdateBatchProgress(id string, processed, successful, failed int) error {
	query := `
		UPDATE lotes 
		SET documentos_procesados = $2, documentos_exitosos = $3, 
			documentos_fallidos = $4, fecha_actualizacion = $5
		WHERE id = $1`

	_, err := r.db.Exec(query, id, processed, successful, failed, time.Now())
	if err != nil {
		return fmt.Errorf("error actualizando progreso del lote: %v", err)
	}

	return nil
}

// FinalizeBatch finaliza un lote
func (r *ComprobanteRepository) FinalizeBatch(id, estado string, fechaFin time.Time) error {
	query := `
		UPDATE lotes 
		SET estado = $2, fecha_fin = $3, fecha_actualizacion = $4
		WHERE id = $1`

	_, err := r.db.Exec(query, id, estado, fechaFin, time.Now())
	if err != nil {
		return fmt.Errorf("error finalizando lote: %v", err)
	}

	return nil
}

// Métodos auxiliares privados

func (r *ComprobanteRepository) insertItem(tx *sql.Tx, comprobanteID string, item models.Item) error {
	query := `
		INSERT INTO items (
			comprobante_id, numero_item, codigo, codigo_sunat, descripcion,
			unidad_medida, cantidad, valor_unitario, precio_unitario,
			descuento_unitario, tipo_afectacion, valor_venta, valor_total
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	var itemID int64
	err := tx.QueryRow(query,
		comprobanteID, item.NumeroItem, item.Codigo, nullString(item.CodigoSUNAT), item.Descripcion,
		item.UnidadMedida, item.Cantidad, item.ValorUnitario, item.PrecioUnitario,
		item.DescuentoUnitario, item.TipoAfectacion, item.ValorVenta, item.ValorTotal,
	).Scan(&itemID)

	if err != nil {
		return err
	}

	// Insertar impuestos del item
	for _, impuesto := range item.ImpuestoItem {
		if err := r.insertImpuestoItem(tx, itemID, impuesto); err != nil {
			return err
		}
	}

	return nil
}

func (r *ComprobanteRepository) insertImpuesto(tx *sql.Tx, comprobanteID string, itemID *int64, impuesto models.Impuesto) error {
	query := `
		INSERT INTO impuestos (
			comprobante_id, item_id, tipo_impuesto, codigo_impuesto,
			base_imponible, tasa, monto_impuesto
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := tx.Exec(query,
		comprobanteID, itemID, impuesto.TipoImpuesto, impuesto.CodigoImpuesto,
		impuesto.BaseImponible, impuesto.Tasa, impuesto.MontoImpuesto,
	)

	return err
}

func (r *ComprobanteRepository) insertImpuestoItem(tx *sql.Tx, itemID int64, impuesto models.ImpuestoItem) error {
	query := `
		INSERT INTO impuestos (
			item_id, tipo_impuesto, codigo_impuesto,
			base_imponible, tasa, monto_impuesto
		) VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := tx.Exec(query,
		itemID, impuesto.TipoImpuesto, impuesto.CodigoImpuesto,
		impuesto.BaseImponible, impuesto.Tasa, impuesto.MontoImpuesto,
	)

	return err
}

func (r *ComprobanteRepository) insertTotales(tx *sql.Tx, comprobanteID string, totales models.Totales) error {
	query := `
		INSERT INTO totales (
			comprobante_id, total_venta_gravada, total_venta_exonerada,
			total_venta_inafecta, total_venta_gratuita, total_descuentos,
			total_anticipos, total_impuestos, total_valor_venta,
			total_precio_venta, redondeo, importe_total
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := tx.Exec(query,
		comprobanteID, totales.TotalVentaGravada, totales.TotalVentaExonerada,
		totales.TotalVentaInafecta, totales.TotalVentaGratuita, totales.TotalDescuentos,
		totales.TotalAnticipos, totales.TotalImpuestos, totales.TotalValorVenta,
		totales.TotalPrecioVenta, totales.Redondeo, totales.ImporteTotal,
	)

	return err
}

func (r *ComprobanteRepository) getItems(comprobanteID string) ([]models.Item, error) {
	query := `
		SELECT id, numero_item, codigo, codigo_sunat, descripcion,
			unidad_medida, cantidad, valor_unitario, precio_unitario,
			descuento_unitario, tipo_afectacion, valor_venta, valor_total
		FROM items WHERE comprobante_id = $1 ORDER BY numero_item`

	rows, err := r.db.Query(query, comprobanteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		var codigoSunat sql.NullString
		var itemIDDB int64

		err := rows.Scan(
			&itemIDDB, &item.NumeroItem, &item.Codigo, &codigoSunat, &item.Descripcion,
			&item.UnidadMedida, &item.Cantidad, &item.ValorUnitario, &item.PrecioUnitario,
			&item.DescuentoUnitario, &item.TipoAfectacion, &item.ValorVenta, &item.ValorTotal,
		)
		if err != nil {
			return nil, err
		}

		if codigoSunat.Valid {
			item.CodigoSUNAT = codigoSunat.String
		}

		// Cargar impuestos del item
		impuestosItem, err := r.getImpuestosItem(itemIDDB)
		if err != nil {
			return nil, err
		}
		item.ImpuestoItem = impuestosItem

		items = append(items, item)
	}

	return items, nil
}

func (r *ComprobanteRepository) getImpuestos(comprobanteID string) ([]models.Impuesto, error) {
	query := `
		SELECT tipo_impuesto, codigo_impuesto, base_imponible, tasa, monto_impuesto
		FROM impuestos 
		WHERE comprobante_id = $1 AND item_id IS NULL`

	rows, err := r.db.Query(query, comprobanteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var impuestos []models.Impuesto
	for rows.Next() {
		var impuesto models.Impuesto
		err := rows.Scan(
			&impuesto.TipoImpuesto, &impuesto.CodigoImpuesto,
			&impuesto.BaseImponible, &impuesto.Tasa, &impuesto.MontoImpuesto,
		)
		if err != nil {
			return nil, err
		}
		impuestos = append(impuestos, impuesto)
	}

	return impuestos, nil
}

func (r *ComprobanteRepository) getImpuestosItem(itemID int64) ([]models.ImpuestoItem, error) {
	query := `
		SELECT tipo_impuesto, codigo_impuesto, base_imponible, tasa, monto_impuesto
		FROM impuestos WHERE item_id = $1`

	rows, err := r.db.Query(query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var impuestos []models.ImpuestoItem
	for rows.Next() {
		var impuesto models.ImpuestoItem
		err := rows.Scan(
			&impuesto.TipoImpuesto, &impuesto.CodigoImpuesto,
			&impuesto.BaseImponible, &impuesto.Tasa, &impuesto.MontoImpuesto,
		)
		if err != nil {
			return nil, err
		}
		impuestos = append(impuestos, impuesto)
	}

	return impuestos, nil
}

func (r *ComprobanteRepository) getTotales(comprobanteID string) (models.Totales, error) {
	query := `
		SELECT total_venta_gravada, total_venta_exonerada, total_venta_inafecta,
			total_venta_gratuita, total_descuentos, total_anticipos,
			total_impuestos, total_valor_venta, total_precio_venta,
			redondeo, importe_total
		FROM totales WHERE comprobante_id = $1`

	var totales models.Totales
	err := r.db.QueryRow(query, comprobanteID).Scan(
		&totales.TotalVentaGravada, &totales.TotalVentaExonerada, &totales.TotalVentaInafecta,
		&totales.TotalVentaGratuita, &totales.TotalDescuentos, &totales.TotalAnticipos,
		&totales.TotalImpuestos, &totales.TotalValorVenta, &totales.TotalPrecioVenta,
		&totales.Redondeo, &totales.ImporteTotal,
	)

	if err == sql.ErrNoRows {
		// Retornar totales vacíos si no existen
		return models.Totales{}, nil
	}

	return totales, err
}

// LogProcess registra un log de proceso
func (r *ComprobanteRepository) LogProcess(comprobanteID, proceso, estado, mensaje string, duracionMs int) error {
	query := `
		INSERT INTO process_log (comprobante_id, proceso, estado, mensaje, duracion_ms)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(query, comprobanteID, proceso, estado, mensaje, duracionMs)
	if err != nil {
		return fmt.Errorf("error registrando log de proceso: %v", err)
	}

	return nil
}

// GetProcessLogs obtiene logs de proceso de un comprobante
func (r *ComprobanteRepository) GetProcessLogs(comprobanteID string) ([]map[string]interface{}, error) {
	query := `
		SELECT proceso, estado, mensaje, fecha_proceso, duracion_ms
		FROM process_log 
		WHERE comprobante_id = $1 
		ORDER BY fecha_proceso DESC`

	rows, err := r.db.Query(query, comprobanteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var proceso, estado, mensaje string
		var fechaProceso time.Time
		var duracionMs sql.NullInt64

		err := rows.Scan(&proceso, &estado, &mensaje, &fechaProceso, &duracionMs)
		if err != nil {
			return nil, err
		}

		log := map[string]interface{}{
			"proceso":       proceso,
			"estado":        estado,
			"mensaje":       mensaje,
			"fecha_proceso": fechaProceso,
		}

		if duracionMs.Valid {
			log["duracion_ms"] = duracionMs.Int64
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// Función auxiliar para manejar strings nullable
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}