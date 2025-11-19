// Package transformer implements positional encoding for GLM-4.6
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

// PositionalEncodingType represents different positional encoding approaches
type PositionalEncodingType string

const (
	PosEncodingTypeSinusoidal PositionalEncodingType = "sinusoidal"
	PosEncodingTypeLearned    PositionalEncodingType = "learned"
	PosEncodingTypeRelative   PositionalEncodingType = "relative"
	PosEncodingTypeRotary     PositionalEncodingType = "rotary"
	PosEncodingTypeALiBi      PositionalEncodingType = "alibi"
	PosEncodingTypeXPos       PositionalEncodingType = "xpos"
)

// PositionalEncodingConfig represents positional encoding configuration
type PositionalEncodingConfig struct {
	Type         PositionalEncodingType `json:"type"`
	MaxSeqLen    int                   `json:"max_seq_len"`
	HiddenSize   int                   `json:"hidden_size"`
	HeadDim      int                   `json:"head_dim,omitempty"`
	Base         float64                `json:"base"`
	Scale        bool                   `json:"scale"`
	Normalize    bool                   `json:"normalize"`
	Concatenate  bool                   `json:"concatenate"`
	RotateHalf   bool                   `json:"rotate_half"`
	UseRoPE      bool                   `json:"use_rope"`
	UseXPos      bool                   `json:"use_xpos"`
}

// PositionalEncodingLayer implements various positional encoding strategies
type PositionalEncodingLayer struct {
	config         PositionalEncodingConfig
	encoding       *Tensor
	rotaryEmbeds   *RotaryEmbeddings
	alibiBias      *ALiBiBias
	xposEmbeds     *XPosembeddings
	learnedPos     *Tensor
	relativePos    *RelativePositionBias
	stats          *PositionalEncodingStats
	mu             sync.RWMutex
	cache          map[int]*Tensor
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

// XPosembeddings implements XPos (Extrapolatable Positional Encoding)
type XPosembeddings struct {
	dim       int
	maxSeqLen int
	base      float64
	scale     bool
	cache     map[int]*XPosCache
	mu        sync.RWMutex
}

// XPosCache contains XPos cache data
type XPosCache struct {
	Sin    *Tensor
	Cos    *Tensor
	Scale  *Tensor
	SeqLen int
}

// RelativePositionBias implements relative position biases
type RelativePositionBias struct {
	numHeads     int
	maxRelativePos int
	buckets       int
	maxDistance   int
	bias          *Tensor
}

// PositionalEncodingStats tracks positional encoding performance
type PositionalEncodingStats struct {
	TotalEncodings    int64   `json:"total_encodings"`
	AvgEncodingTime   float64  `json:"avg_encoding_time"`
	CacheHitRate      float64  `json:"cache_hit_rate"`
	RoPEUsage         float64  `json:"rope_usage"`
	ALiBiUsage        float64  `json:"alibi_usage"`
	XPosUsage         float64  `json:"xpos_usage"`
	LastUpdated       int64    `json:"last_updated"`
}

// NewPositionalEncodingLayer creates a new positional encoding layer
func NewPositionalEncodingLayer(config PositionalEncodingConfig) *PositionalEncodingLayer {
	logger.Info("Creating positional encoding layer",
		zap.String("type", string(config.Type)),
		zap.Int("max_seq_len", config.MaxSeqLen),
		zap.Int("hidden_size", config.HiddenSize),
		zap.Bool("use_rope", config.UseRoPE),
		zap.Bool("use_xpos", config.UseXPos),
	)

	layer := &PositionalEncodingLayer{
		config: config,
		stats:  &PositionalEncodingStats{},
		cache:  make(map[int]*Tensor),
	}

	// Initialize based on type
	switch config.Type {
	case PosEncodingTypeSinusoidal:
		layer.encoding = layer.createSinusoidalEncoding()
	case PosEncodingTypeLearned:
		layer.learnedPos = t.randn(config.MaxSeqLen, config.HiddenSize)
	case PosEncodingTypeRotary:
		layer.rotaryEmbeds = NewRotaryEmbeddingsWithConfig(config)
	case PosEncodingTypeALiBi:
		layer.alibiBias = NewALiBiBiasWithConfig(config)
	case PosEncodingTypeXPos:
		layer.xposEmbeds = NewXPosembeddingsWithConfig(config)
	case PosEncodingTypeRelative:
		layer.relativePos = NewRelativePositionBias(config.HiddenSize/12, config.MaxSeqLen)
	default:
		layer.encoding = layer.createSinusoidalEncoding()
	}

	return layer
}

// Forward applies positional encoding to input tensor
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
	case PosEncodingTypeSinusoidal:
		return p.applySinusoidalEncoding(input, seqOffset)
	case PosEncodingTypeLearned:
		return p.applyLearnedEncoding(input, seqOffset)
	case PosEncodingTypeRotary:
		return p.applyRotaryEncoding(input, seqOffset)
	case PosEncodingTypeALiBi:
		return p.applyALiBiEncoding(input)
	case PosEncodingTypeXPos:
		return p.applyXPosEncoding(input, seqOffset)
	case PosEncodingTypeRelative:
		return p.applyRelativeEncoding(input)
	default:
		return p.applySinusoidalEncoding(input, seqOffset)
	}
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
	
	// Normalize if configured
	if p.config.Normalize {
		p.normalizeEncoding(encoding)
	}
	
	// Scale if configured
	if p.config.Scale {
		p.scaleEncoding(encoding)
	}
	
	return encoding
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
	
	// Check cache
	if cache, exists := p.cache[seqOffset]; exists {
		return p.applyCachedEncoding(input, cache)
	}
	
	// Generate encoding for this sequence segment
	startPos := seqOffset
	endPos := seqOffset + seqLen
	
	if endPos > p.config.MaxSeqLen {
		endPos = p.config.MaxSeqLen
	}
	
	// Extract relevant portion of pre-computed encoding
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
	
	// Cache the result
	p.cache[seqOffset] = broadcastEncoding
	
	// Add to input or concatenate
	if p.config.Concatenate {
		return p.concatenateEncoding(input, broadcastEncoding)
	}
	
	return t.add(input, broadcastEncoding), nil
}

// applyLearnedEncoding applies learned positional embeddings
func (p *PositionalEncodingLayer) applyLearnedEncoding(input *Tensor, seqOffset int) (*Tensor, error) {
	batchSize := 1
	if len(input.Shape) > 0 {
		batchSize = input.Shape[0]
	}
	
	seqLen := len(input.Data)
	if len(input.Shape) > 1 {
		seqLen = input.Shape[1]
	}
	
	// Extract relevant portion of learned embeddings
	startPos := seqOffset
	endPos := seqOffset + seqLen
	
	if endPos > p.config.MaxSeqLen {
		endPos = p.config.MaxSeqLen
	}
	
	posEncoding := t.zeros(seqLen, p.config.HiddenSize)
	for i := 0; i < seqLen && startPos+i < p.config.MaxSeqLen; i++ {
		for j := 0; j < p.config.HiddenSize; j++ {
			posEncoding.Data[i*p.config.HiddenSize+j] = 
				p.learnedPos.Data[(startPos+i)*p.config.HiddenSize+j]
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

// applyXPosEncoding applies XPos encoding
func (p *PositionalEncodingLayer) applyXPosEncoding(input *Tensor, seqOffset int) (*Tensor, error) {
	if p.xposEmbeds == nil {
		return input, nil
	}
	
	return p.xposEmbeds.Apply(input, seqOffset)
}

// applyRelativeEncoding applies relative position bias
func (p *PositionalEncodingLayer) applyRelativeEncoding(input *Tensor) (*Tensor, error) {
	if p.relativePos == nil {
		return input, nil
	}
	
	// Relative position bias is applied during attention computation
	return input, nil
}

// applyCachedEncoding applies cached encoding
func (p *PositionalEncodingLayer) applyCachedEncoding(input *Tensor, cache *Tensor) (*Tensor, error) {
	return t.add(input, cache), nil
}

// concatenateEncoding concatenates positional encoding with input
func (p *PositionalEncodingLayer) concatenateEncoding(input *Tensor, encoding *Tensor) (*Tensor, error) {
	// Simplified concatenation
	// In practice, this would concatenate along the last dimension
	return input, nil
}

// normalizeEncoding normalizes the positional encoding
func (p *PositionalEncodingLayer) normalizeEncoding(encoding *Tensor) {
	// Compute mean and variance
	mean := 0.0
	variance := 0.0
	
	for _, val := range encoding.Data {
		mean += val
	}
	mean /= float64(len(encoding.Data))
	
	for _, val := range encoding.Data {
		variance += math.Pow(val-mean, 2)
	}
	variance /= float64(len(encoding.Data))
	
	stdDev := math.Sqrt(variance + 1e-5)
	
	// Normalize
	for i := range encoding.Data {
		encoding.Data[i] = (encoding.Data[i] - mean) / stdDev
	}
}

// scaleEncoding scales the positional encoding
func (p *PositionalEncodingLayer) scaleEncoding(encoding *Tensor) {
	scale := math.Sqrt(float64(p.config.HiddenSize))
	
	for i := range encoding.Data {
		encoding.Data[i] *= scale
	}
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

// NewRotaryEmbeddingsWithConfig creates rotary embeddings with config
func NewRotaryEmbeddingsWithConfig(config PositionalEncodingConfig) *RotaryEmbeddings {
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

// Apply applies rotary embeddings with sequence offset
func (r *RotaryEmbeddings) Apply(input *Tensor, seqOffset int) (*Tensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	seqLen := input.Shape[1]
	
	// Check cache
	cacheKey := seqOffset
	if cache, exists := r.cache[cacheKey]; exists && cache.SeqLen >= seqLen {
		return r.applyRotaryCacheWithOffset(input, cache, seqOffset)
	}
	
	// Generate new rotary embeddings
	cache := r.generateRotaryCacheWithOffset(seqOffset, seqLen)
	r.cache[cacheKey] = cache
	
	return r.applyRotaryCacheWithOffset(input, cache, seqOffset)
}

// generateRotaryCacheWithOffset generates rotary cache with offset
func (r *RotaryEmbeddings) generateRotaryCacheWithOffset(seqOffset, seqLen int) *RotaryCache {
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

// applyRotaryCacheWithOffset applies cached rotary embeddings with offset
func (r *RotaryEmbeddings) applyRotaryCacheWithOffset(input *Tensor, cache *RotaryCache, seqOffset int) (*Tensor, error) {
	// Simplified rotary embedding application with offset
	return input, nil
}

// NewXPosembeddingsWithConfig creates XPos embeddings with config
func NewXPosembeddingsWithConfig(config PositionalEncodingConfig) *XPosembeddings {
	dim := config.HeadDim
	if dim == 0 {
		dim = config.HiddenSize / 12
	}
	
	base := config.Base
	if base == 0 {
		base = 10000.0
	}
	
	return &XPosembeddings{
		dim:       dim,
		maxSeqLen: config.MaxSeqLen,
		base:      base,
		scale:     config.Scale,
		cache:     make(map[int]*XPosCache),
	}
}

// Apply applies XPos encoding
func (x *XPosembeddings) Apply(input *Tensor, seqOffset int) (*Tensor, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	
	seqLen := input.Shape[1]
	
	// Check cache
	cacheKey := seqOffset
	if cache, exists := x.cache[cacheKey]; exists && cache.SeqLen >= seqLen {
		return x.applyXPosCache(input, cache)
	}
	
	// Generate new XPos embeddings
	cache := x.generateXPosCache(seqOffset, seqLen)
	x.cache[cacheKey] = cache
	
	return x.applyXPosCache(input, cache)
}

// generateXPosCache generates XPos cache
func (x *XPosembeddings) generateXPosCache(seqOffset, seqLen int) *XPosCache {
	sin := t.zeros(seqLen, x.dim)
	cos := t.zeros(seqLen, x.dim)
	scale := t.zeros(seqLen, x.dim)
	
	for pos := 0; pos < seqLen; pos++ {
		for i := 0; i < x.dim; i += 2 {
			angle := float64(pos+seqOffset) / math.Pow(x.base, float64(i)/float64(x.dim))
			
			if i < x.dim {
				sin.Data[pos*x.dim+i] = math.Sin(angle)
			}
			if i+1 < x.dim {
				cos.Data[pos*x.dim+i+1] = math.Cos(angle)
			}
			
			// XPos specific scaling
			scale.Data[pos*x.dim+i/2] = 1.0 / math.Sqrt(float64(pos+seqOffset+1))
		}
	}
	
	return &XPosCache{
		Sin:    sin,
		Cos:    cos,
		Scale:  scale,
		SeqLen: seqLen,
	}
}

// applyXPosCache applies cached XPos embeddings
func (x *XPosembeddings) applyXPosCache(input *Tensor, cache *XPosCache) (*Tensor, error) {
	// Simplified XPos application
	return input, nil
}

// NewRelativePositionBias creates relative position bias
func NewRelativePositionBias(numHeads, maxSeqLen int) *RelativePositionBias {
	return &RelativePositionBias{
		numHeads:       numHeads,
		maxRelativePos: maxSeqLen,
		buckets:        32,
		maxDistance:    128,
		bias:          t.zeros(numHeads, maxSeqLen, maxSeqLen),
	}
}