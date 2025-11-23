// Package relational provides PostgreSQL relational database implementations
package relational

import (
	"database/sql"
)

// InitAllSchemas creates all database schemas
func InitAllSchemas(db *sql.DB) error {
	schemas := []string{
		mcpSchema,
		knowledgeSchema,
		projectSchema,
		templateSchema,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}

const mcpSchema = `
	CREATE TABLE IF NOT EXISTS mcps (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		stack VARCHAR(50) NOT NULL,
		path VARCHAR(500),
		features JSONB,
		context JSONB,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_mcps_stack ON mcps(stack);
	CREATE INDEX IF NOT EXISTS idx_mcps_name ON mcps(name);
	CREATE INDEX IF NOT EXISTS idx_mcps_created_at ON mcps(created_at);
`

const knowledgeSchema = `
	CREATE TABLE IF NOT EXISTS knowledge (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		documents JSONB,
		embeddings JSONB,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_knowledge_name ON knowledge(name);
	CREATE INDEX IF NOT EXISTS idx_knowledge_version ON knowledge(version);
	CREATE INDEX IF NOT EXISTS idx_knowledge_created_at ON knowledge(created_at);
`

const projectSchema = `
	CREATE TABLE IF NOT EXISTS projects (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		mcp_id VARCHAR(36) NOT NULL,
		stack VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		FOREIGN KEY (mcp_id) REFERENCES mcps(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_projects_mcp_id ON projects(mcp_id);
	CREATE INDEX IF NOT EXISTS idx_projects_stack ON projects(stack);
	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects(created_at);
`

const templateSchema = `
	CREATE TABLE IF NOT EXISTS templates (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		stack VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		variables JSONB,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_templates_stack ON templates(stack);
	CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);
	CREATE INDEX IF NOT EXISTS idx_templates_created_at ON templates(created_at);
`
