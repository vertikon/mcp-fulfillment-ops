-- Migration: Create fulfillment tables
-- Description: Cria tabelas para entidades de fulfillment operations

-- Tabela de Inbound Shipments (Recebimentos)
CREATE TABLE IF NOT EXISTS inbound_shipments (
    id VARCHAR(255) PRIMARY KEY,
    reference_id VARCHAR(255) NOT NULL,
    origin VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_inbound_reference_id ON inbound_shipments(reference_id);
CREATE INDEX idx_inbound_status ON inbound_shipments(status);
CREATE INDEX idx_inbound_idempotency_key ON inbound_shipments(idempotency_key);

-- Tabela de Fulfillment Orders (Expedições)
CREATE TABLE IF NOT EXISTS fulfillment_orders (
    id VARCHAR(255) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    customer VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    shipped_at TIMESTAMP
);

CREATE INDEX idx_fulfillment_order_id ON fulfillment_orders(order_id);
CREATE INDEX idx_fulfillment_status ON fulfillment_orders(status);
CREATE INDEX idx_fulfillment_idempotency_key ON fulfillment_orders(idempotency_key);

-- Tabela de Transfer Orders (Transferências)
CREATE TABLE IF NOT EXISTS transfer_orders (
    id VARCHAR(255) PRIMARY KEY,
    location_from VARCHAR(255) NOT NULL,
    location_to VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_transfer_status ON transfer_orders(status);
CREATE INDEX idx_transfer_idempotency_key ON transfer_orders(idempotency_key);

-- Tabela de Return Orders (Devoluções)
CREATE TABLE IF NOT EXISTS return_orders (
    id VARCHAR(255) PRIMARY KEY,
    original_order_id VARCHAR(255) NOT NULL,
    reason VARCHAR(500),
    status VARCHAR(50) NOT NULL,
    items JSONB NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_return_original_order_id ON return_orders(original_order_id);
CREATE INDEX idx_return_status ON return_orders(status);
CREATE INDEX idx_return_idempotency_key ON return_orders(idempotency_key);

-- Tabela de Cycle Count Tasks (Contagens Cíclicas)
CREATE TABLE IF NOT EXISTS cycle_count_tasks (
    id VARCHAR(255) PRIMARY KEY,
    location VARCHAR(255) NOT NULL,
    skus JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    counted_items JSONB NOT NULL DEFAULT '[]',
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_cycle_count_location ON cycle_count_tasks(location);
CREATE INDEX idx_cycle_count_status ON cycle_count_tasks(status);
CREATE INDEX idx_cycle_count_idempotency_key ON cycle_count_tasks(idempotency_key);

