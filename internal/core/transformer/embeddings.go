// Package transformer implements embeddings and positional encoding for GLM-4.6
package transformer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// EmbeddingType represents different embedding types
type EmbeddingType string

const (
	EmbeddingTypeToken      EmbeddingType = "token"
	EmbeddingTypePositional EmbeddingType = "position"
	EmbeddingTypeRotary     EmbeddingType = "rotary"
	EmbeddingTypeALiBi      EmbeddingType = "alibi"
	EmbeddingTypeLearned    EmbeddingType = "learned"
)

// PositionalEncodingType represents different positional encoding types
type PositionalEncodingType string

const (
	PosEncodingSinusoidal PositionalEncodingType = "sinusoidal"
	PosEncodingLearned    PositionalEncodingType = "learned"
	PosEncodingRelative   PositionalEncodingType = "relative"
	PosEncodingRotary     PositionalEncodingType = "rotary"
	PosEncodingALiBi      PositionalEncodingType = "alibi"
)

// EmbeddingConfig represents embedding layer configuration
type EmbeddingConfig struct {
	VocabSize        int                 `json:"vocab_size"`
	HiddenSize       int                 `json:"hidden_size"`
	EmbeddingType    EmbeddingType        `json:"embedding_type"`
	MaxPosition      int                 `json:"max_position"`
	PaddingIdx       int                 `json:"padding_idx"`
	FreezeEmbeddings bool                `json:"freeze_embeddings"`
	LearnEmbeddings  bool                `json:"learn_embeddings"`
	Dropout          float64             `json:"dropout"`
}

// PositionalEncodingConfig represents positional encoding configuration
type PositionalEncodingConfig struct {
	Type         PositionalEncodingType `json:"type"`
	MaxSeqLen    int                   `json:"max_seq_len"`
	HiddenSize   int                   `json:"hidden_size"`
	Base         float64                `json:"base"`
	Scale        bool                   `json:"scale"`
	Normalize    bool                   `json:"normalize"`
	UseRoPE      bool                   `json:"use_rope"`
	HeadDim      int                   `json:"head_dim,omitempty"`
}

// EmbeddingLayer implements token embeddings
type EmbeddingLayer struct {
	config        EmbeddingConfig
	weight        *Tensor
	bias          *Tensor
	norm          *LayerNorm
	stats         *EmbeddingStats
	mu            sync.RWMutex
}

// PositionalEncodingLayer implements positional encoding
type PositionalEncodingLayer struct {
	config        PositionalEncodingConfig
	encoding      *Tensor
	rotaryEmbeds  *RotaryEmbeddings
	alibiBias     *ALiBiBias
	stats         *PositionalEncodingStats
	mu            sync.RWMutex
}

// RotaryEmbeddings implements rotary positional embeddings (RoPE)
type RotaryEmbeddings struct {
	dim          int
	maxSeqLen    int
	base         float64
	scale        bool
	cache        map[int]*RotaryCache
	sinCosCache  *Tensor
	mu           sync.RWMutex
}

// RotaryCache contains cached rotary embeddings
type RotaryCache struct {
	Sin *Tensor
	Cos *Tensor
	SeqLen int
}

// ALiBiBias implements attention linear bias (ALiBi)
type ALiBiBias struct {
	numHeads   int
	maxSeqLen  int
	slope      []float64
	bias       *Tensor
}

// EmbeddingStats tracks embedding layer statistics
type EmbeddingStats struct {
	TotalLookups   int64   `json:"total_lookups"`
	AvgLookupTime  float64  `json:"avg_lookup_time"`
	CacheHitRate   float64  `json:"cache_hit_rate"`
	EmbeddingNorm  float64  `json:"embedding_norm"`
	LastUpdated    int64    `json:"last_updated"`
}

// PositionalEncodingStats tracks positional encoding statistics
type PositionalEncodingStats struct {
	TotalEncodings    int64   `json:"total_encodings"`
	AvgEncodingTime   float64  `json:"avg_encoding_time"`
	RoPEUsage        float64  `json:"rope_usage"`
	ALiBiUsage       float64  `json:"alibi_usage"`
	LastUpdated       int64    `json:"last_updated"`
}

// NewEmbeddingLayer creates a new embedding layer
func NewEmbeddingLayer(config EmbeddingConfig) *EmbeddingLayer {
	logger.Info("Creating embedding layer",
		zap.Int("vocab_size", config.VocabSize),
		zap.Int("hidden_size", config.HiddenSize),
		zap.String("type", string(config.EmbeddingType)),
	)

	// Initialize embedding weights
	weight := t.randn(config.VocabSize, config.HiddenSize)
	
	// Initialize bias if not using positional bias
	bias := t.zeros(config.HiddenSize)
	if config.PaddingIdx >= 0 {
		// Zero out padding token embedding
		for i := 0; i < config.HiddenSize; i++ {
			weight.Data[config.PaddingIdx*config.HiddenSize+i] = 0.0
		}
	}

	layer := &EmbeddingLayer{
		config: config,
		weight: weight,
		bias:   bias,
		stats:  &EmbeddingStats{},
	}

	// Initialize layer normalization if needed
	if config.LearnEmbeddings {
		layer.norm = NewLayerNorm(config.HiddenSize, 1e-5)
	}

	return layer
}

// Forward performs embedding lookup
func (e *EmbeddingLayer) Forward(ctx context.Context, input *Tensor) (*Tensor, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	start := time.Now()
	defer func() {
		e.updateStats(time.Since(start))
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch e.config.EmbeddingType {
	case EmbeddingTypeToken:
		return e.tokenEmbedding(input)
	default:
		return e.tokenEmbedding(input)
	}
}

// tokenEmbedding performs standard token embedding lookup
func (e *EmbeddingLayer) tokenEmbedding(input *Tensor) (*Tensor, error) {
	// Simplified embedding lookup
	// In practice, this would use efficient indexing operations
	
	// For demonstration, create dummy output
	batchSize := 1
	if len(input.Shape) > 0 {
		batchSize = input.Shape[0]
	}
	
	seqLen := len(input.Data)
	if len(input.Shape) > 1 {
		seqLen = input.Shape[1]
	}
	
	output := t.zeros(batchSize, seqLen, e.config.HiddenSize)
	
	// Apply layer normalization if configured
	if e.norm != nil {
		return e.norm.Forward(output)
	}
	
	return output, nil
}

// updateStats updates embedding statistics
func (e *EmbeddingLayer) updateStats(computationTime time.Duration) {
	e.stats.TotalLookups++
	e.stats.AvgLookupTime = 
		(e.stats.AvgLookupTime*float64(e.stats.TotalLookups-1) + 
		 computationTime.Seconds()) / float64(e.stats.TotalLookups)
	e.stats.LastUpdated = time.Now().Unix()
}

// GetStats returns embedding statistics
func (e *EmbeddingLayer) GetStats() EmbeddingStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return *e.stats
}

// NewPositionalEncodingLayer creates a new positional encoding layer
func NewPositionalEncodingLayer(config PositionalEncodingConfig) *PositionalEncodingLayer {
	logger.Info("Creating positional encoding layer",
		zap.String("type", string(config.Type)),
		zap.Int("max_seq_len", config.MaxSeqLen),
		zap.Int("hidden_size", config.HiddenSize),
	)

	layer := &PositionalEncodingLayer{
		config: config,
		stats:  &PositionalEncodingStats{},
	}

	switch config.Type {
	case PosEncodingSinusoidal:
		layer.encoding = layer.createSinusoidalEncoding()
	case PosEncodingRotary:
		layer.rotaryEmbeds = NewRotaryEmbeddingsFull(config)
	case PosEncodingALiBi:
		layer.alibiBias = NewALiBiBiasFull(config)
	default:
		layer.encoding = layer.createSinusoidalEncoding()
	}

	return layer
}

// createSinusoidalEncoding creates sinusoidal positional encoding
func (p *PositionalEncodingLayer) createSinusoidalEncoding() *Tensor {
	encoding := t.zeros(p.config.MaxSeqLen, p.config.HiddenSize)
	
	for pos := 0; pos < p.config.MaxSeqLen; pos++ {
		for i := 0; i < p.config.HiddenSize; i += 2 {
			angle := float64(pos) / math.Pow(p.config.Base, float64(i)/float64(p.config.HiddenSize))
			
			if i < p.config.HiddenSize {
				encoding.Data[pos*p.config.HiddenSize+i] = math.Sin(angle)
			}
			if i+1 < p.config.HiddenSize {
				encoding.Data[pos*p.config.HiddenSize+i+1] = math.Cos(angle)
			}
		}
	}
	
	return encoding
}

// Forward applies positional encoding to input
func (p *PositionalEncodingLayer) Forward(ctx context.Context, input *Tensor, seqOffset int) (*Tensor, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch p.config.Type {
	case PosEncodingSinusoidal:
		return p.applySinusoidalEncoding(input, seqOffset)
	case PosEncodingRotary:
		return p.applyRotaryEncoding(input, seqOffset)
	case PosEncodingALiBi:
		return p.applyALiBiEncoding(input)
	default:
		return p.applySinusoidalEncoding(input, seqOffset)
	}
}

// applySinusoidalEncoding applies sinusoidal positional encoding
func (p *PositionalEncodingLayer) applySinusoidalEncoding(input *Tensor, seqOffset int) (*Tensor, error) {
	batchSize := 1
	if len(input.Shape) > 0 {
		batchSize = input.Shape[0]
	}
	
	seqLen := len(input.Data)
	if len(input.Shape) > 1 {
		seqLen = input.Shape[1]
	}
	
	// Extract relevant portion of encoding
	startPos := seqOffset
	endPos := seqOffset + seqLen
	
	if endPos > p.config.MaxSeqLen {
		endPos = p.config.MaxSeqLen
	}
	
	// Create positional encoding for this sequence
	posEncoding := t.zeros(seqLen, p.config.HiddenSize)
	for i := 0; i < seqLen && startPos+i < p.config.MaxSeqLen; i++ {
		for j := 0; j < p.config.HiddenSize; j++ {
			posEncoding.Data[i*p.config.HiddenSize+j] = 
				p.encoding.Data[(startPos+i)*p.config.HiddenSize+j]
		}
	}
	
	// Broadcast to batch dimension
	broadcastEncoding := t.zeros(batchSize, seqLen, p.config.HiddenSize)
	for b := 0; b < batchSize; b++ {
		for i := 0; i < seqLen; i++ {
			for j := 0; j < p.config.HiddenSize; j++ {
				broadcastEncoding.Data[b*seqLen*p.config.HiddenSize+i*p.config.HiddenSize+j] = 
					posEncoding.Data[i*p.config.HiddenSize+j]
			}
		}
	}
	
	// Add to input
	return t.add(input, broadcastEncoding), nil
}

// applyRotaryEncoding applies rotary positional encoding
func (p *PositionalEncodingLayer) applyRotaryEncoding(input *Tensor, seqOffset int) (*Tensor, error) {
	if p.rotaryEmbeds == nil {
		return input, nil
	}
	
	return p.rotaryEmbeds.Apply(input, seqOffset)
}

// applyALiBiEncoding applies ALiBi bias
func (p *PositionalEncodingLayer) applyALiBiEncoding(input *Tensor) (*Tensor, error) {
	if p.alibiBias == nil {
		return input, nil
	}
	
	// ALiBi is applied during attention computation
	return input, nil
}

// updateStats updates positional encoding statistics
func (p *PositionalEncodingLayer) updateStats(computationTime time.Duration) {
	p.stats.TotalEncodings++
	p.stats.AvgEncodingTime = 
		(p.stats.AvgEncodingTime*float64(p.stats.TotalEncodings-1) + 
		 computationTime.Seconds()) / float64(p.stats.TotalEncodings)
	p.stats.LastUpdated = time.Now().Unix()
}

// GetStats returns positional encoding statistics
func (p *PositionalEncodingLayer) GetStats() PositionalEncodingStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return *p.stats
}

// NewRotaryEmbeddingsFull creates full rotary embeddings
func NewRotaryEmbeddingsFull(config PositionalEncodingConfig) *RotaryEmbeddings {
	headDim := config.HeadDim
	if headDim == 0 {
		headDim = config.HiddenSize / 12 // Default to 12 heads
	}
	
	base := config.Base
	if base == 0 {
		base = 10000.0
	}
	
	return &RotaryEmbeddings{
		dim:       headDim,
		maxSeqLen: config.MaxSeqLen,
		base:      base,
		scale:     config.Scale,
		cache:     make(map[int]*RotaryCache),
	}
}

// Apply applies rotary embeddings to input tensor
func (r *RotaryEmbeddings) Apply(input *Tensor, seqOffset int) (*Tensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	seqLen := input.Shape[1]
	
	// Check cache
	cacheKey := seqOffset
	if cache, exists := r.cache[cacheKey]; exists && cache.SeqLen >= seqLen {
		return r.applyRotaryCache(input, cache)
	}
	
	// Generate new rotary embeddings
	cache := r.generateRotaryCache(seqOffset, seqLen)
	r.cache[cacheKey] = cache
	
	return r.applyRotaryCache(input, cache)
}

// generateRotaryCache generates rotary embedding cache
func (r *RotaryEmbeddings) generateRotaryCache(seqOffset, seqLen int) *RotaryCache {
	sin := t.zeros(seqLen, r.dim)
	cos := t.zeros(seqLen, r.dim)
	
	for pos := 0; pos < seqLen; pos++ {
		for i := 0; i < r.dim; i += 2 {
			angle := float64(pos+seqOffset) / math.Pow(r.base, float64(i)/float64(r.dim))
			
			if i < r.dim {
				sin.Data[pos*r.dim+i] = math.Sin(angle)
			}
			if i+1 < r.dim {
				cos.Data[pos*r.dim+i+1] = math.Cos(angle)
			}
		}
	}
	
	return &RotaryCache{
		Sin:    sin,
		Cos:    cos,
		SeqLen: seqLen,
	}
}

// applyRotaryCache applies cached rotary embeddings
func (r *RotaryEmbeddings) applyRotaryCache(input *Tensor, cache *RotaryCache) (*Tensor, error) {
	// Simplified rotary embedding application
	// In practice, this would rotate the attention keys and queries
	return input, nil
}

// NewALiBiBiasFull creates full ALiBi bias
func NewALiBiBiasFull(config PositionalEncodingConfig) *ALiBiBias {
	numHeads := 12 // Default assumption
	slope := make([]float64, numHeads)
	
	for i := 0; i < numHeads; i++ {
		slope[i] = math.Pow(2.0, -8.0/float64(numHeads)*float64(i))
	}
	
	bias := t.zeros(numHeads, config.MaxSeqLen, config.MaxSeqLen)
	
	// Generate ALiBi bias matrix
	for head := 0; head < numHeads; head++ {
		for i := 0; i < config.MaxSeqLen; i++ {
			for j := 0; j < config.MaxSeqLen; j++ {
				if j > i {
					bias.Data[head*config.MaxSeqLen*config.MaxSeqLen+i*config.MaxSeqLen+j] = 
						slope[head] * float64(j-i)
				}
			}
		}
	}
	
	return &ALiBiBias{
		numHeads:  numHeads,
		maxSeqLen: config.MaxSeqLen,
		slope:     slope,
		bias:      bias,
	}
}