package repository

import (
	"database/sql"
	"facturacion_sunat_api_go/internal/config"
	"fmt"
	"time"
)

// InitDatabase inicializa y configura la base de datos PostgreSQL
func InitDatabase(config config.DatabaseConfig) (*sql.DB, error) {
	// Construir DSN para PostgreSQL
	dsn := buildPostgreSQLDSN(config)

	// Abrir conexión a PostgreSQL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexión a PostgreSQL: %v", err)
	}

	// Configurar parámetros de conexión
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Verificar conexión
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL: %v", err)
	}

	// Crear esquema y tablas si no existen
	if err := createSchema(db, config.Schema); err != nil {
		return nil, fmt.Errorf("error creando esquema: %v", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creando tablas: %v", err)
	}

	// Insertar datos iniciales
	if err := insertInitialData(db); err != nil {
		return nil, fmt.Errorf("error insertando datos iniciales: %v", err)
	}

	return db, nil
}

// buildPostgreSQLDSN construye la cadena de conexión para PostgreSQL
func buildPostgreSQLDSN(config config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)
}

// createSchema crea el esquema si no existe
func createSchema(db *sql.DB, schema string) error {
	if schema == "" {
		schema = "public" // esquema por defecto
	}

	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("error creando esquema %s: %v", schema, err)
	}

	// Establecer search_path
	searchPathQuery := fmt.Sprintf("SET search_path TO %s, public", schema)
	if _, err := db.Exec(searchPathQuery); err != nil {
		return fmt.Errorf("error estableciendo search_path: %v", err)
	}

	return nil
}

// createTables crea las tablas necesarias
func createTables(db *sql.DB) error {
	queries := []string{
		createComprobantesTable(),
		createItemsTable(),
		createImpuestosTable(),
		createTotalesTable(),
		createProcessLogTable(),
		createConfiguracionTable(),
		createCertificadosTable(),
		createLotesTable(),
		createIndices(),
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("error ejecutando query: %v", err)
		}
	}

	return nil
}

func createComprobantesTable() string {
	return `
	CREATE TABLE IF NOT EXISTS comprobantes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tipo INTEGER NOT NULL CHECK (tipo IN (1, 2, 3, 4)),
		serie VARCHAR(10) NOT NULL,
		numero VARCHAR(20) NOT NULL,
		fecha_emision TIMESTAMP WITH TIME ZONE NOT NULL,
		fecha_vencimiento TIMESTAMP WITH TIME ZONE,
		tipo_moneda VARCHAR(3) NOT NULL DEFAULT 'PEN',
		
		-- Emisor
		emisor_ruc VARCHAR(11) NOT NULL,
		emisor_razon_social VARCHAR(500) NOT NULL,
		emisor_nombre_comercial VARCHAR(500),
		emisor_direccion VARCHAR(500) NOT NULL,
		emisor_distrito VARCHAR(100) NOT NULL,
		emisor_provincia VARCHAR(100) NOT NULL,
		emisor_departamento VARCHAR(100) NOT NULL,
		emisor_pais VARCHAR(3) NOT NULL DEFAULT 'PE',
		emisor_telefono VARCHAR(20),
		emisor_email VARCHAR(100),
		
		-- Receptor
		receptor_tipo_documento VARCHAR(2) NOT NULL,
		receptor_numero_documento VARCHAR(20) NOT NULL,
		receptor_razon_social VARCHAR(500) NOT NULL,
		receptor_direccion VARCHAR(500),
		receptor_email VARCHAR(100),
		
		-- Totales calculados
		total_valor_venta DECIMAL(15,2) DEFAULT 0 CHECK (total_valor_venta >= 0),
		total_impuestos DECIMAL(15,2) DEFAULT 0 CHECK (total_impuestos >= 0),
		total_precio_venta DECIMAL(15,2) DEFAULT 0 CHECK (total_precio_venta >= 0),
		importe_total DECIMAL(15,2) DEFAULT 0 CHECK (importe_total >= 0),
		
		-- Estado del proceso
		estado_proceso INTEGER DEFAULT 1 CHECK (estado_proceso BETWEEN 1 AND 7),
		
		-- Datos XML y SUNAT
		xml_generado TEXT,
		xml_firmado TEXT,
		archivo_zip BYTEA,
		ticket_sunat VARCHAR(100),
		cdr_sunat BYTEA,
		estado_sunat VARCHAR(50),
		observaciones TEXT,
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		fecha_actualizacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		usuario_creacion VARCHAR(100),
		
		-- Constraints
		CONSTRAINT uk_comprobante_serie_numero UNIQUE(tipo, serie, numero),
		CONSTRAINT chk_emisor_ruc CHECK (LENGTH(emisor_ruc) = 11 AND emisor_ruc ~ '^[0-9]{11}$')
	);
	`
}

func createItemsTable() string {
	return `
	CREATE TABLE IF NOT EXISTS items (
		id BIGSERIAL PRIMARY KEY,
		comprobante_id UUID NOT NULL,
		numero_item INTEGER NOT NULL CHECK (numero_item > 0),
		codigo VARCHAR(50) NOT NULL,
		codigo_sunat VARCHAR(50),
		descripcion VARCHAR(1000) NOT NULL,
		unidad_medida VARCHAR(10) NOT NULL,
		cantidad DECIMAL(15,4) NOT NULL CHECK (cantidad > 0),
		valor_unitario DECIMAL(15,4) NOT NULL CHECK (valor_unitario >= 0),
		precio_unitario DECIMAL(15,4) NOT NULL CHECK (precio_unitario >= 0),
		descuento_unitario DECIMAL(15,4) DEFAULT 0 CHECK (descuento_unitario >= 0),
		tipo_afectacion INTEGER NOT NULL CHECK (tipo_afectacion BETWEEN 10 AND 40),
		valor_venta DECIMAL(15,2) NOT NULL CHECK (valor_venta >= 0),
		valor_total DECIMAL(15,2) NOT NULL CHECK (valor_total >= 0),
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_items_comprobante FOREIGN KEY (comprobante_id) 
			REFERENCES comprobantes(id) ON DELETE CASCADE,
		CONSTRAINT uk_item_comprobante_numero UNIQUE(comprobante_id, numero_item)
	);`
}

func createImpuestosTable() string {
	return `
	CREATE TABLE IF NOT EXISTS impuestos (
		id BIGSERIAL PRIMARY KEY,
		comprobante_id UUID,
		item_id BIGINT,
		tipo_impuesto VARCHAR(10) NOT NULL,
		codigo_impuesto VARCHAR(10) NOT NULL,
		base_imponible DECIMAL(15,2) NOT NULL CHECK (base_imponible >= 0),
		tasa DECIMAL(8,4) NOT NULL CHECK (tasa >= 0),
		monto_impuesto DECIMAL(15,2) NOT NULL CHECK (monto_impuesto >= 0),
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_impuestos_comprobante FOREIGN KEY (comprobante_id) 
			REFERENCES comprobantes(id) ON DELETE CASCADE,
		CONSTRAINT fk_impuestos_item FOREIGN KEY (item_id) 
			REFERENCES items(id) ON DELETE CASCADE,
		CONSTRAINT chk_impuesto_referencia CHECK (
			(comprobante_id IS NOT NULL AND item_id IS NULL) OR
			(comprobante_id IS NULL AND item_id IS NOT NULL)
		)
	);`
}

func createTotalesTable() string {
	return `
	CREATE TABLE IF NOT EXISTS totales (
		comprobante_id UUID PRIMARY KEY,
		total_venta_gravada DECIMAL(15,2) DEFAULT 0 CHECK (total_venta_gravada >= 0),
		total_venta_exonerada DECIMAL(15,2) DEFAULT 0 CHECK (total_venta_exonerada >= 0),
		total_venta_inafecta DECIMAL(15,2) DEFAULT 0 CHECK (total_venta_inafecta >= 0),
		total_venta_gratuita DECIMAL(15,2) DEFAULT 0 CHECK (total_venta_gratuita >= 0),
		total_descuentos DECIMAL(15,2) DEFAULT 0 CHECK (total_descuentos >= 0),
		total_anticipos DECIMAL(15,2) DEFAULT 0 CHECK (total_anticipos >= 0),
		total_impuestos DECIMAL(15,2) DEFAULT 0 CHECK (total_impuestos >= 0),
		total_valor_venta DECIMAL(15,2) DEFAULT 0 CHECK (total_valor_venta >= 0),
		total_precio_venta DECIMAL(15,2) DEFAULT 0 CHECK (total_precio_venta >= 0),
		redondeo DECIMAL(15,2) DEFAULT 0,
		importe_total DECIMAL(15,2) DEFAULT 0 CHECK (importe_total >= 0),
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		fecha_actualizacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_totales_comprobante FOREIGN KEY (comprobante_id) 
			REFERENCES comprobantes(id) ON DELETE CASCADE
	);`
}

func createProcessLogTable() string {
	return `
	CREATE TABLE IF NOT EXISTS process_log (
		id BIGSERIAL PRIMARY KEY,
		comprobante_id UUID NOT NULL,
		proceso VARCHAR(50) NOT NULL,
		estado VARCHAR(20) NOT NULL,
		mensaje VARCHAR(1000),
		detalle_error TEXT,
		fecha_proceso TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		duracion_ms INTEGER CHECK (duracion_ms >= 0),
		
		CONSTRAINT fk_process_log_comprobante FOREIGN KEY (comprobante_id) 
			REFERENCES comprobantes(id) ON DELETE CASCADE
	);`
}

func createConfiguracionTable() string {
	return `
	CREATE TABLE IF NOT EXISTS configuracion (
		clave VARCHAR(100) PRIMARY KEY,
		valor TEXT NOT NULL,
		descripcion VARCHAR(500),
		tipo VARCHAR(20) DEFAULT 'string' CHECK (tipo IN ('string', 'integer', 'decimal', 'boolean', 'json')),
		es_activo BOOLEAN DEFAULT true,
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		fecha_actualizacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
}

func createCertificadosTable() string {
	return `
	CREATE TABLE IF NOT EXISTS certificados (
		id BIGSERIAL PRIMARY KEY,
		nombre VARCHAR(100) NOT NULL,
		ruta_archivo VARCHAR(500) NOT NULL,
		password_hash VARCHAR(255),
		subject VARCHAR(500),
		issuer VARCHAR(500),
		fecha_inicio TIMESTAMP WITH TIME ZONE,
		fecha_expiracion TIMESTAMP WITH TIME ZONE,
		fingerprint_sha1 VARCHAR(64),
		fingerprint_sha256 VARCHAR(128),
		algoritmo VARCHAR(50),
		longitud_clave INTEGER,
		es_valido_sunat BOOLEAN DEFAULT false,
		es_activo BOOLEAN DEFAULT true,
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		fecha_actualizacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		usuario_creacion VARCHAR(100),
		
		CONSTRAINT uk_certificado_nombre UNIQUE(nombre),
		CONSTRAINT chk_fechas_certificado CHECK (fecha_inicio < fecha_expiracion)
	);`
}

func createLotesTable() string {
	return `
	CREATE TABLE IF NOT EXISTS lotes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		descripcion VARCHAR(500),
		total_documentos INTEGER DEFAULT 0 CHECK (total_documentos >= 0),
		documentos_procesados INTEGER DEFAULT 0 CHECK (documentos_procesados >= 0),
		documentos_exitosos INTEGER DEFAULT 0 CHECK (documentos_exitosos >= 0),
		documentos_fallidos INTEGER DEFAULT 0 CHECK (documentos_fallidos >= 0),
		estado VARCHAR(30) DEFAULT 'PENDIENTE' CHECK (estado IN ('PENDIENTE', 'PROCESANDO', 'COMPLETADO', 'COMPLETADO_CON_ERRORES', 'ERROR')),
		fecha_inicio TIMESTAMP WITH TIME ZONE,
		fecha_fin TIMESTAMP WITH TIME ZONE,
		configuracion_proceso JSONB,
		
		-- Auditoría
		fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		fecha_actualizacion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		usuario_creacion VARCHAR(100),
		
		CONSTRAINT chk_documentos_procesados CHECK (documentos_procesados <= total_documentos),
		CONSTRAINT chk_documentos_exitosos CHECK (documentos_exitosos <= documentos_procesados),
		CONSTRAINT chk_documentos_fallidos CHECK (documentos_fallidos <= documentos_procesados),
		CONSTRAINT chk_suma_documentos CHECK (documentos_exitosos + documentos_fallidos = documentos_procesados)
	);`
}

func createIndices() string {
	return `
	-- Índices para optimizar consultas
	CREATE INDEX IF NOT EXISTS idx_comprobantes_fecha_emision ON comprobantes(fecha_emision);
	CREATE INDEX IF NOT EXISTS idx_comprobantes_estado_proceso ON comprobantes(estado_proceso);
	CREATE INDEX IF NOT EXISTS idx_comprobantes_emisor_ruc ON comprobantes(emisor_ruc);
	CREATE INDEX IF NOT EXISTS idx_comprobantes_receptor_documento ON comprobantes(receptor_numero_documento);
	CREATE INDEX IF NOT EXISTS idx_comprobantes_ticket_sunat ON comprobantes(ticket_sunat);
	CREATE INDEX IF NOT EXISTS idx_comprobantes_fecha_creacion ON comprobantes(fecha_creacion);
	
	CREATE INDEX IF NOT EXISTS idx_items_comprobante_id ON items(comprobante_id);
	CREATE INDEX IF NOT EXISTS idx_items_codigo ON items(codigo);
	
	CREATE INDEX IF NOT EXISTS idx_impuestos_comprobante_id ON impuestos(comprobante_id);
	CREATE INDEX IF NOT EXISTS idx_impuestos_item_id ON impuestos(item_id);
	CREATE INDEX IF NOT EXISTS idx_impuestos_tipo ON impuestos(tipo_impuesto, codigo_impuesto);
	
	CREATE INDEX IF NOT EXISTS idx_process_log_comprobante_id ON process_log(comprobante_id);
	CREATE INDEX IF NOT EXISTS idx_process_log_fecha_proceso ON process_log(fecha_proceso);
	CREATE INDEX IF NOT EXISTS idx_process_log_proceso_estado ON process_log(proceso, estado);
	
	CREATE INDEX IF NOT EXISTS idx_certificados_es_activo ON certificados(es_activo);
	CREATE INDEX IF NOT EXISTS idx_certificados_fecha_expiracion ON certificados(fecha_expiracion);
	CREATE INDEX IF NOT EXISTS idx_certificados_es_valido_sunat ON certificados(es_valido_sunat);
	
	CREATE INDEX IF NOT EXISTS idx_lotes_estado ON lotes(estado);
	CREATE INDEX IF NOT EXISTS idx_lotes_fecha_creacion ON lotes(fecha_creacion);
	`
}

func insertInitialData(db *sql.DB) error {
	// Insertar configuraciones iniciales
	configs := []struct {
		clave       string
		valor       string
		descripcion string
		tipo        string
	}{
		{"empresa_ruc", "20123456789", "RUC de la empresa", "string"},
		{"empresa_razon_social", "MI EMPRESA SAC", "Razón social de la empresa", "string"},
		{"serie_factura_default", "F001", "Serie por defecto para facturas", "string"},
		{"serie_boleta_default", "B001", "Serie por defecto para boletas", "string"},
		{"igv_rate", "18.00", "Tasa de IGV", "decimal"},
		{"auto_calculate_totals", "true", "Calcular totales automáticamente", "boolean"},
		{"auto_send_sunat", "false", "Enviar automáticamente a SUNAT", "boolean"},
		{"backup_xml", "true", "Respaldar archivos XML", "boolean"},
		{"log_level", "INFO", "Nivel de logging", "string"},
		{"max_batch_size", "100", "Tamaño máximo de lote", "integer"},
		{"connection_timeout", "60", "Timeout de conexión SUNAT (segundos)", "integer"},
		{"retry_attempts", "3", "Intentos de reenvío automático", "integer"},
	}

	for _, config := range configs {
		query := `
		INSERT INTO configuracion (clave, valor, descripcion, tipo) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (clave) DO NOTHING`
		
		_, err := db.Exec(query, config.clave, config.valor, config.descripcion, config.tipo)
		if err != nil {
			return fmt.Errorf("error insertando configuración %s: %v", config.clave, err)
		}
	}

	return nil
}

// GetDatabaseStats obtiene estadísticas de la base de datos
func GetDatabaseStats(db *sql.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Contar comprobantes por estado
	query := `
	SELECT estado_proceso, COUNT(*) as total 
	FROM comprobantes 
	GROUP BY estado_proceso
	ORDER BY estado_proceso`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas: %v", err)
	}
	defer rows.Close()

	estados := make(map[int]int)
	for rows.Next() {
		var estado, total int
		if err := rows.Scan(&estado, &total); err != nil {
			return nil, err
		}
		estados[estado] = total
	}
	stats["comprobantes_por_estado"] = estados

	// Total de comprobantes
	var totalComprobantes int
	err = db.QueryRow("SELECT COUNT(*) FROM comprobantes").Scan(&totalComprobantes)
	if err != nil {
		return nil, err
	}
	stats["total_comprobantes"] = totalComprobantes

	// Comprobantes por tipo
	queryTipos := `
	SELECT tipo, COUNT(*) as total 
	FROM comprobantes 
	GROUP BY tipo
	ORDER BY tipo`
	
	rows2, err := db.Query(queryTipos)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	tipos := make(map[int]int)
	for rows2.Next() {
		var tipo, total int
		if err := rows2.Scan(&tipo, &total); err != nil {
			return nil, err
		}
		tipos[tipo] = total
	}
	stats["comprobantes_por_tipo"] = tipos

	// Estadísticas de la última semana
	queryRecientes := `
	SELECT 
		COUNT(*) as total_semana,
		COUNT(*) FILTER (WHERE estado_proceso = 5) as enviados_semana,
		COUNT(*) FILTER (WHERE estado_proceso = 6) as aceptados_semana
	FROM comprobantes 
	WHERE fecha_creacion >= CURRENT_DATE - INTERVAL '7 days'`
	
	var totalSemana, enviadosSemana, aceptadosSemana int
	err = db.QueryRow(queryRecientes).Scan(&totalSemana, &enviadosSemana, &aceptadosSemana)
	if err != nil {
		return nil, err
	}
	
	stats["estadisticas_semana"] = map[string]int{
		"total":     totalSemana,
		"enviados":  enviadosSemana,
		"aceptados": aceptadosSemana,
	}

	// Tamaño de la base de datos (PostgreSQL específico)
	var dbSize int64
	querySize := `
	SELECT pg_database_size(current_database())`
	
	err = db.QueryRow(querySize).Scan(&dbSize)
	if err == nil {
		stats["database_size_bytes"] = dbSize
	}

	// Información de conexiones
	var activeConnections, maxConnections int
	queryConn := `
	SELECT 
		(SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active') as active,
		(SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_conn`
	
	err = db.QueryRow(queryConn).Scan(&activeConnections, &maxConnections)
	if err == nil {
		stats["conexiones"] = map[string]int{
			"activas": activeConnections,
			"maximas": maxConnections,
		}
	}

	return stats, nil
}

// CleanupOldRecords limpia registros antiguos
func CleanupOldRecords(db *sql.DB, daysToKeep int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %v", err)
	}
	defer tx.Rollback()

	// Limpiar logs de proceso antiguos
	queryLogs := `
	DELETE FROM process_log 
	WHERE fecha_proceso < CURRENT_TIMESTAMP - INTERVAL '%d days'`
	
	result, err := tx.Exec(fmt.Sprintf(queryLogs, daysToKeep))
	if err != nil {
		return fmt.Errorf("error limpiando process_log: %v", err)
	}
	
	logsDeleted, _ := result.RowsAffected()

	// Limpiar comprobantes con errores muy antiguos (solo los rechazados definitivamente)
	queryComprobantes := `
	DELETE FROM comprobantes 
	WHERE fecha_creacion < CURRENT_TIMESTAMP - INTERVAL '%d days' 
	AND estado_proceso IN (6, 7)` // Solo rechazados o con error
	
	result2, err := tx.Exec(fmt.Sprintf(queryComprobantes, daysToKeep*3)) // Más tiempo para comprobantes
	if err != nil {
		return fmt.Errorf("error limpiando comprobantes antiguos: %v", err)
	}
	
	comprobantesDeleted, _ := result2.RowsAffected()

	// Limpiar lotes completados antiguos
	queryLotes := `
	DELETE FROM lotes 
	WHERE fecha_creacion < CURRENT_TIMESTAMP - INTERVAL '%d days' 
	AND estado IN ('COMPLETADO', 'COMPLETADO_CON_ERRORES')`
	
	result3, err := tx.Exec(fmt.Sprintf(queryLotes, daysToKeep*2))
	if err != nil {
		return fmt.Errorf("error limpiando lotes antiguos: %v", err)
	}
	
	lotesDeleted, _ := result3.RowsAffected()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error confirmando limpieza: %v", err)
	}

	fmt.Printf("Limpieza completada: %d logs, %d comprobantes, %d lotes eliminados\n", 
		logsDeleted, comprobantesDeleted, lotesDeleted)

	return nil
}

// BackupDatabase crea un respaldo de la base de datos PostgreSQL
func BackupDatabase(db *sql.DB, backupPath string) error {
	// Para PostgreSQL, se debe usar pg_dump externamente
	// Esta función prepara la información necesaria
	
	var dbName, host, port, user string
	query := `
	SELECT 
		current_database() as db_name,
		inet_server_addr() as host,
		inet_server_port() as port,
		current_user as user`
	
	err := db.QueryRow(query).Scan(&dbName, &host, &port, &user)
	if err != nil {
		return fmt.Errorf("error obteniendo información de conexión: %v", err)
	}

	// Retornar información para que el llamador ejecute pg_dump
	fmt.Printf("Para crear backup ejecutar:\n")
	fmt.Printf("pg_dump -h %s -p %s -U %s -d %s -f %s\n", 
		host, port, user, dbName, backupPath)
	
	return nil
}

// CreatePartitions crea particiones para tablas grandes (opcional)
func CreatePartitions(db *sql.DB) error {
	// Particionar process_log por fecha
	partitionQuery := `
	-- Crear tabla particionada para process_log si no existe
	CREATE TABLE IF NOT EXISTS process_log_partitioned (
		LIKE process_log INCLUDING ALL
	) PARTITION BY RANGE (fecha_proceso);
	
	-- Crear particiones mensuales automáticamente
	CREATE TABLE IF NOT EXISTS process_log_2025_01 
	PARTITION OF process_log_partitioned 
	FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
	
	CREATE TABLE IF NOT EXISTS process_log_2025_02 
	PARTITION OF process_log_partitioned 
	FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');`
	
	_, err := db.Exec(partitionQuery)
	if err != nil {
		return fmt.Errorf("error creando particiones: %v", err)
	}
	
	return nil
}

// OptimizeDatabase ejecuta tareas de mantenimiento
func OptimizeDatabase(db *sql.DB) error {
	maintenanceQueries := []string{
		"VACUUM ANALYZE comprobantes;",
		"VACUUM ANALYZE items;",
		"VACUUM ANALYZE process_log;",
		"REINDEX TABLE comprobantes;",
		"UPDATE pg_stat_user_tables SET n_tup_ins=0, n_tup_upd=0, n_tup_del=0;",
	}

	for _, query := range maintenanceQueries {
		if _, err := db.Exec(query); err != nil {
			fmt.Printf("Warning: Error en mantenimiento %s: %v\n", query, err)
			// No retornamos error para que continúe con otras tareas
		}
	}

	return nil
}