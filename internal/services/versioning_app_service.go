package services

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/versioning/data"
	"github.com/vertikon/mcp-fulfillment-ops/internal/versioning/knowledge"
	"github.com/vertikon/mcp-fulfillment-ops/internal/versioning/models"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VersioningAppService provides application-level versioning services
type VersioningAppService struct {
	knowledgeVersioning knowledge.KnowledgeVersioning
	modelRegistry       models.ModelRegistry
	modelVersioning     models.ModelVersioning
	abTesting           models.ABTesting
	modelDeployment     models.ModelDeployment
	dataVersioning      data.DataVersioning
	schemaMigration     data.SchemaMigrationEngine
	dataLineage         data.DataLineageTracker
	dataQuality         data.DataQuality
	logger              *zap.Logger
}

// NewVersioningAppService creates a new versioning app service
func NewVersioningAppService() *VersioningAppService {
	// Initialize components
	kv := knowledge.NewInMemoryKnowledgeVersioning()
	mr := models.NewInMemoryModelRegistry()
	mv := models.NewInMemoryModelVersioning(mr)
	ab := models.NewInMemoryABTesting()
	md := models.NewInMemoryModelDeployment()
	dv := data.NewInMemoryDataVersioning()
	sm := data.NewInMemorySchemaMigrationEngine()
	dl := data.NewInMemoryDataLineageTracker()
	dq := data.NewInMemoryDataQuality()

	return &VersioningAppService{
		knowledgeVersioning: kv,
		modelRegistry:       mr,
		modelVersioning:     mv,
		abTesting:           ab,
		modelDeployment:     md,
		dataVersioning:      dv,
		schemaMigration:     sm,
		dataLineage:         dl,
		dataQuality:         dq,
		logger:              logger.WithContext(context.Background()),
	}
}

// GetKnowledgeVersioning returns knowledge versioning service
func (s *VersioningAppService) GetKnowledgeVersioning() knowledge.KnowledgeVersioning {
	return s.knowledgeVersioning
}

// GetModelRegistry returns model registry service
func (s *VersioningAppService) GetModelRegistry() models.ModelRegistry {
	return s.modelRegistry
}

// GetModelVersioning returns model versioning service
func (s *VersioningAppService) GetModelVersioning() models.ModelVersioning {
	return s.modelVersioning
}

// GetABTesting returns A/B testing service
func (s *VersioningAppService) GetABTesting() models.ABTesting {
	return s.abTesting
}

// GetModelDeployment returns model deployment service
func (s *VersioningAppService) GetModelDeployment() models.ModelDeployment {
	return s.modelDeployment
}

// GetDataVersioning returns data versioning service
func (s *VersioningAppService) GetDataVersioning() data.DataVersioning {
	return s.dataVersioning
}

// GetSchemaMigration returns schema migration service
func (s *VersioningAppService) GetSchemaMigration() data.SchemaMigrationEngine {
	return s.schemaMigration
}

// GetDataLineage returns data lineage service
func (s *VersioningAppService) GetDataLineage() data.DataLineageTracker {
	return s.dataLineage
}

// GetDataQuality returns data quality service
func (s *VersioningAppService) GetDataQuality() data.DataQuality {
	return s.dataQuality
}
