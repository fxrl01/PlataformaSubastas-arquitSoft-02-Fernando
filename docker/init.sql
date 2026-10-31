-- =========================================================================
-- ESQUEMA DDL RELACIONAL: LIVEBID PLATAFORMA DE SUBASTAS EN TIEMPO REAL
-- =========================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Tabla de Usuarios e Identidad
CREATE TABLE IF NOT EXISTS usuarios (
    id VARCHAR(64) PRIMARY KEY,
    nombre VARCHAR(120) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    rol VARCHAR(30) NOT NULL CHECK (rol IN ('POSTOR', 'MARTILLERO', 'VENDEDOR', 'ADMIN')),
    estado VARCHAR(30) NOT NULL DEFAULT 'PENDIENTE'
        CHECK (estado IN ('PENDIENTE', 'ACTIVO', 'SUSPENDIDO', 'BLOQUEADO')),
    kyc_aprobado BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS estado VARCHAR(30) NOT NULL DEFAULT 'PENDIENTE';
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- 2. Tabla de Subastas y Lotes
CREATE TABLE IF NOT EXISTS subastas (
    id VARCHAR(64) PRIMARY KEY,
    titulo VARCHAR(255) NOT NULL,
    descripcion TEXT,
    precio_base NUMERIC(12, 2) NOT NULL CHECK (precio_base > 0),
    incremento_minimo NUMERIC(12, 2) NOT NULL CHECK (incremento_minimo > 0),
    precio_actual NUMERIC(12, 2) NOT NULL,
    postor_lider_id VARCHAR(64) REFERENCES usuarios(id),
    estado VARCHAR(30) NOT NULL CHECK (estado IN ('PROGRAMADA', 'ACTIVA', 'EN_CONTEO_MARTILLO', 'ADJUDICADA', 'DESIERTA', 'CANCELADA')),
    fin_programado TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices de alto rendimiento para sala en vivo
CREATE INDEX IF NOT EXISTS idx_subastas_estado ON subastas(estado);
CREATE INDEX IF NOT EXISTS idx_subastas_fin ON subastas(fin_programado);

-- 3. Tabla de Depósitos en Custodia (Escrow)
CREATE TABLE IF NOT EXISTS garantias (
    id VARCHAR(64) PRIMARY KEY,
    subasta_id VARCHAR(64) NOT NULL REFERENCES subastas(id) ON DELETE CASCADE,
    postor_id VARCHAR(64) NOT NULL REFERENCES usuarios(id),
    monto_retenido NUMERIC(12, 2) NOT NULL CHECK (monto_retenido > 0),
    autorizacion_bancaria_id VARCHAR(100) NOT NULL,
    estado VARCHAR(30) NOT NULL CHECK (estado IN ('RETENIDA', 'CAPTURADA_COBRO', 'LIBERADA')),
    score_confianza_ia INT NOT NULL CHECK (score_confianza_ia BETWEEN 0 AND 100),
    fecha_retencion TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_garantias_subasta_postor ON garantias(subasta_id, postor_id, estado);

-- 4. Tabla de Historial Inmutable de Pujas
CREATE TABLE IF NOT EXISTS pujas (
    id VARCHAR(64) PRIMARY KEY,
    subasta_id VARCHAR(64) NOT NULL REFERENCES subastas(id) ON DELETE CASCADE,
    postor_id VARCHAR(64) NOT NULL REFERENCES usuarios(id),
    monto NUMERIC(12, 2) NOT NULL CHECK (monto > 0),
    correlativo INT NOT NULL,
    timestamp_oficial TIMESTAMP WITH TIME ZONE NOT NULL,
    anti_sniping_disparado BOOLEAN DEFAULT FALSE,
    hash_auditoria VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_subasta_correlativo UNIQUE (subasta_id, correlativo)
);

CREATE INDEX IF NOT EXISTS idx_pujas_subasta_correlativo ON pujas(subasta_id, correlativo DESC);

-- 5. Tabla de Auditoría Legal y Trazabilidad Forense
CREATE TABLE IF NOT EXISTS logs_auditoria (
    id BIGSERIAL PRIMARY KEY,
    subasta_id VARCHAR(64) NOT NULL,
    evento VARCHAR(60) NOT NULL,
    payload JSONB NOT NULL,
    hash_anterior VARCHAR(64),
    hash_actual VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_logs_subasta_fecha ON logs_auditoria(subasta_id, created_at DESC);

-- Datos iniciales de prueba (Seed Data)
INSERT INTO usuarios (id, nombre, email, rol, kyc_aprobado)
VALUES 
    ('USR-VIP-01', 'Fernando Olivares', 'fernando@livebid.pe', 'POSTOR', TRUE),
    ('USR-RIESGO_ALTO', 'Postor Anónimo', 'riesgo@livebid.pe', 'POSTOR', FALSE),
    ('MART-01', 'Martillero Oficial', 'martillero@livebid.pe', 'MARTILLERO', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO subastas (id, titulo, precio_base, incremento_minimo, precio_actual, estado, fin_programado)
VALUES 
    ('SUB-2026-001', 'Vehículo Toyota Hilux 2022 4x4', 45000.00, 1000.00, 45000.00, 'ACTIVA', CURRENT_TIMESTAMP + INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;
