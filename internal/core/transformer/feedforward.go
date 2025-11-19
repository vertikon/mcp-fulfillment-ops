// Package transformer implements feed-forward networks for GLM-4.6
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

// ActivationType represents different activation functions
type ActivationType string

const (
	ActivationGELU      ActivationType = "gelu"
	ActivationReLU      ActivationType = "relu"
	ActivationSwiGLU    ActivationType = "swiglu"
	ActivationGeGLU     ActivationType = "geglu"
	ActivationSiLU      ActivationType = "silu"
	ActivationTanh      ActivationType = "tanh"
	ActivationSigmoid   ActivationType = "sigmoid"
)

// FeedForwardConfig represents feed-forward network configuration
type FeedForwardConfig struct {
	HiddenSize     int           `json:"hidden_size"`
	IntermediateSize int          `json:"intermediate_size"`
	Activation      ActivationType `json:"activation"`
	Dropout         float64       `json:"dropout"`
	UseBias         bool          `json:"use_bias"`
	UseSwiGLU       bool          `json:"use_swiglu"`
	ExpertSize      int           `json:"expert_size,omitempty"`
	NumExperts      int           `json:"num_experts,omitempty"`
	TopK            int           `json:"top_k,omitempty"`
}

// FeedForwardNetwork implements position-wise feed-forward networks
type FeedForwardNetwork struct {
	config           FeedForwardConfig
	activationFunc   ActivationFunction
	gateWeights      *Tensor
	gateBias         *Tensor
	upWeights        *Tensor
	upBias           *Tensor
	downWeights      *Tensor
	downBias         *Tensor
	expertWeights    []*Tensor
	expertBiases     []*Tensor
	routerWeights    *Tensor
	routerBias       *Tensor
	mu               sync.RWMutex
	stats            *FeedForwardStats
}

// MoELayer implements mixture of experts
type MoELayer struct {
	config         FeedForwardConfig
	experts        []*FeedForwardNetwork
	router         *Router
	topK           int
	loadBalancing   bool
	stats          *MoEStats
}

// Router implements expert routing for MoE
type Router struct {
	weights *Tensor
	bias    *Tensor
	topK    int
	stats   *RouterStats
}

// FeedForwardStats tracks feed-forward network statistics
type FeedForwardStats struct {
	TotalForward   int64   `json:"total_forward"`
	AvgForwardTime float64  `json:"avg_forward_time"`
	ActivationUtil float64   `json:"activation_utilization"`
	SparsityRatio float64   `json:"sparsity_ratio"`
	LastUpdated   int64     `json:"last_updated"`
}

// MoEStats tracks mixture of experts statistics
type MoEStats struct {
	TotalCalls      int64             `json:"total_calls"`
	ExpertUsage     map[int]int64     `json:"expert_usage"`
	LoadBalance     float64           `json:"load_balance"`
	RoutingEntropy  float64           `json:"routing_entropy"`
	AvgForwardTime float64           `json:"avg_forward_time"`
	LastUpdated     int64             `json:"last_updated"`
}

// RouterStats tracks router statistics
type RouterStats struct {
	TotalRoutes     int64   `json:"total_routes"`
	AvgRoutingTime  float64 `json:"avg_routing_time"`
	LoadBalanceCost float64 `json:"load_balance_cost"`
	LastUpdated     int64    `json:"last_updated"`
}

// ActivationFunction represents different activation functions
type ActivationFunction interface {
	Forward(input *Tensor) *Tensor
	Backward(input, grad *Tensor) *Tensor
	Name() string
}

// NewFeedForwardNetwork creates a new feed-forward network
func NewFeedForwardNetwork(config FeedForwardConfig) *FeedForwardNetwork {
	logger.Info("Creating feed-forward network",
		zap.Int("hidden_size", config.HiddenSize),
		zap.Int("intermediate_size", config.IntermediateSize),
		zap.String("activation", string(config.Activation)),
		zap.Bool("use_swiglu", config.UseSwiGLU),
	)

	ffn := &FeedForwardNetwork{
		config:         config,
		activationFunc: NewActivationFunction(config.Activation),
		stats:          &FeedForwardStats{},
	}

	// Initialize weights based on configuration
	if config.UseSwiGLU || config.Activation == ActivationSwiGLU {
		ffn.initSwiGLUWeights()
	} else {
		ffn.initStandardWeights()
	}

	// Initialize MoE if configured
	if config.NumExperts > 0 {
		ffn.initMoEWeights()
	}

	return ffn
}

// Forward performs forward pass through feed-forward network
func (ffn *FeedForwardNetwork) Forward(ctx context.Context, input *Tensor) (*Tensor, error) {
	ffn.mu.RLock()
	defer ffn.mu.RUnlock()

	start := time.Now()
	defer func() {
		ffn.updateStats(time.Since(start))
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Handle MoE routing
	if ffn.config.NumExperts > 0 {
		return ffn.forwardMoE(ctx, input)
	}

	// Standard feed-forward
	return ffn.forwardStandard(ctx, input)
}

// forwardStandard performs standard feed-forward computation
func (ffn *FeedForwardNetwork) forwardStandard(ctx context.Context, input *Tensor) (*Tensor, error) {
	// First linear transformation
	hidden := t.matmul(input, ffn.gateWeights)
	if ffn.gateBias != nil {
		hidden = t.add(hidden, ffn.gateBias)
	}

	// Activation function
	activated := ffn.activationFunc.Forward(hidden)

	// Apply dropout if configured
	if ffn.config.Dropout > 0 {
		activated = ffn.applyDropout(activated, ffn.config.Dropout)
	}

	// Second linear transformation
	output := t.matmul(activated, ffn.downWeights)
	if ffn.downBias != nil {
		output = t.add(output, ffn.downBias)
	}

	return output, nil
}

// forwardMoE performs mixture of experts computation
func (ffn *FeedForwardNetwork) forwardMoE(ctx context.Context, input *Tensor) (*Tensor, error) {
	// Router determines which experts to use
	router, err := ffn.router.Forward(input)
	if err != nil {
		return nil, fmt.Errorf("routing error: %w", err)
	}

	// Select top-k experts
	selectedExperts := ffn.selectTopKExperts(router)

	// Combine expert outputs
	output := ffn.combineExperts(input, selectedExperts)

	return output, nil
}

// initStandardWeights initializes standard feed-forward weights
func (ffn *FeedForwardNetwork) initStandardWeights() {
	ffn.gateWeights = t.randn(ffn.config.HiddenSize, ffn.config.IntermediateSize)
	ffn.gateBias = t.zeros(ffn.config.IntermediateSize)
	ffn.downWeights = t.randn(ffn.config.IntermediateSize, ffn.config.HiddenSize)
	ffn.downBias = t.zeros(ffn.config.HiddenSize)
}

// initSwiGLUWeights initializes SwiGLU-specific weights
func (ffn *FeedForwardNetwork) initSwiGLUWeights() {
	// SwiGLU uses separate gate and up projections
	ffn.gateWeights = t.randn(ffn.config.HiddenSize, ffn.config.IntermediateSize)
	ffn.upWeights = t.randn(ffn.config.HiddenSize, ffn.config.IntermediateSize)
	ffn.downWeights = t.randn(ffn.config.IntermediateSize, ffn.config.HiddenSize)
	ffn.gateBias = t.zeros(ffn.config.IntermediateSize)
	ffn.upBias = t.zeros(ffn.config.IntermediateSize)
	ffn.downBias = t.zeros(ffn.config.HiddenSize)
}

// initMoEWeights initializes mixture of experts weights
func (ffn *FeedForwardNetwork) initMoEWeights() {
	ffn.expertWeights = make([]*Tensor, ffn.config.NumExperts)
	ffn.expertBiases = make([]*Tensor, ffn.config.NumExperts)

	for i := 0; i < ffn.config.NumExperts; i++ {
		ffn.expertWeights[i] = t.randn(ffn.config.HiddenSize, ffn.config.IntermediateSize)
		ffn.expertBiases[i] = t.zeros(ffn.config.IntermediateSize)
	}

	// Router weights
	ffn.routerWeights = t.randn(ffn.config.HiddenSize, ffn.config.NumExperts)
	ffn.routerBias = t.zeros(ffn.config.NumExperts)
}

// applyDropout applies dropout during training
func (ffn *FeedForwardNetwork) applyDropout(input *Tensor, dropoutRate float64) *Tensor {
	// Simplified dropout implementation
	// In practice, this would randomly zero out elements
	return input
}

// selectTopKExperts selects top-k experts based on router scores
func (ffn *FeedForwardNetwork) selectTopKExperts(router *Tensor) []int {
	// Simplified top-k selection
	// In practice, this would select the k highest scoring experts
	topK := ffn.config.TopK
	if topK == 0 {
		topK = 2 // Default
	}

	selected := make([]int, 0, topK)
	for i := 0; i < topK && i < ffn.config.NumExperts; i++ {
		selected = append(selected, i)
	}

	return selected
}

// combineExperts combines outputs from selected experts
func (ffn *FeedForwardNetwork) combineExperts(input *Tensor, selectedExperts []int) *Tensor {
	// Simplified expert combination
	// In practice, this would compute weighted average of expert outputs
	return t.matmul(input, ffn.downWeights)
}

// updateStats updates feed-forward statistics
func (ffn *FeedForwardNetwork) updateStats(computationTime time.Duration) {
	ffn.stats.TotalForward++
	ffn.stats.AvgForwardTime = 
		(ffn.stats.AvgForwardTime*float64(ffn.stats.TotalForward-1) + 
		 computationTime.Seconds()) / float64(ffn.stats.TotalForward)
	ffn.stats.LastUpdated = time.Now().Unix()
}

// GetStats returns feed-forward network statistics
func (ffn *FeedForwardNetwork) GetStats() FeedForwardStats {
	ffn.mu.RLock()
	defer ffn.mu.RUnlock()
	return *ffn.stats
}

// NewMoELayer creates a new mixture of experts layer
func NewMoELayer(config FeedForwardConfig) *MoELayer {
	logger.Info("Creating MoE layer",
		zap.Int("num_experts", config.NumExperts),
		zap.Int("top_k", config.TopK),
	)

	moe := &MoELayer{
		config:       config,
		experts:      make([]*FeedForwardNetwork, config.NumExperts),
		topK:         config.TopK,
		loadBalancing: true,
		stats: &MoEStats{
			ExpertUsage: make(map[int]int64),
		},
	}

	// Create individual experts
	for i := 0; i < config.NumExperts; i++ {
		expertConfig := config
		expertConfig.NumExperts = 0 // Disable MoE in individual experts
		moe.experts[i] = NewFeedForwardNetwork(expertConfig)
	}

	// Initialize router
	moe.router = NewRouter(config.HiddenSize, config.NumExperts, config.TopK)

	return moe
}

// Forward performs MoE forward pass
func (moe *MoELayer) Forward(ctx context.Context, input *Tensor) (*Tensor, error) {
	start := time.Now()
	defer func() {
		moe.updateStats(time.Since(start))
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Route input to experts
	routing, err := moe.router.Route(input)
	if err != nil {
		return nil, fmt.Errorf("routing error: %w", err)
	}

	// Execute selected experts
	expertOutputs := make([]*Tensor, len(routing.Experts))
	for i, expertIdx := range routing.Experts {
		output, err := moe.experts[expertIdx].Forward(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("expert %d error: %w", expertIdx, err)
		}
		expertOutputs[i] = output
		moe.stats.ExpertUsage[expertIdx]++
	}

	// Combine expert outputs
	output := moe.combineExpertOutputs(expertOutputs, routing.Weights)

	return output, nil
}

// combineExpertOutputs combines outputs from multiple experts
func (moe *MoELayer) combineExpertOutputs(outputs []*Tensor, weights []float64) *Tensor {
	// Simplified weighted combination
	if len(outputs) == 0 {
		return outputs[0]
	}

	// Weighted average
	result := t.scale(outputs[0], weights[0])
	for i := 1; i < len(outputs); i++ {
		weighted := t.scale(outputs[i], weights[i])
		result = t.add(result, weighted)
	}

	return result
}

// updateStats updates MoE statistics
func (moe *MoELayer) updateStats(computationTime time.Duration) {
	moe.stats.TotalCalls++
	moe.stats.AvgForwardTime = 
		(moe.stats.AvgForwardTime*float64(moe.stats.TotalCalls-1) + 
		 computationTime.Seconds()) / float64(moe.stats.TotalCalls)
	moe.stats.LastUpdated = time.Now().Unix()
}

// GetStats returns MoE statistics
func (moe *MoELayer) GetStats() MoEStats {
	return *moe.stats
}

// NewRouter creates a new expert router
func NewRouter(inputSize, numExperts, topK int) *Router {
	return &Router{
		weights: t.randn(inputSize, numExperts),
		bias:    t.zeros(numExperts),
		topK:    topK,
		stats:   &RouterStats{},
	}
}

// Route routes inputs to experts
func (r *Router) Route(input *Tensor) (*RoutingResult, error) {
	// Compute routing scores
	scores := t.matmul(input, r.weights)
	if r.bias != nil {
		scores = t.add(scores, r.bias)
	}

	// Apply softmax
	probs := t.softmax(scores, -1)

	// Select top-k experts
	experts, weights := r.selectTopK(probs)

	return &RoutingResult{
		Experts: experts,
		Weights: weights,
		Scores:  probs,
	}, nil
}

// selectTopK selects top-k experts based on probabilities
func (r *Router) selectTopK(probs *Tensor) ([]int, []float64) {
	// Simplified top-k selection
	topK := r.topK
	if topK == 0 {
		topK = 2
	}

	experts := make([]int, 0, topK)
	weights := make([]float64, 0, topK)

	for i := 0; i < topK && i < len(probs.Data); i++ {
		experts = append(experts, i)
		weights = append(weights, probs.Data[i])
	}

	return experts, weights
}

// RoutingResult contains routing information
type RoutingResult struct {
	Experts []int     `json:"experts"`
	Weights []float64 `json:"weights"`
	Scores  *Tensor   `json:"scores"`
}

// NewActivationFunction creates an activation function
func NewActivationFunction(activationType ActivationType) ActivationFunction {
	switch activationType {
	case ActivationGELU:
		return &GELU{}
	case ActivationReLU:
		return &ReLU{}
	case ActivationSwiGLU:
		return &SwiGLU{}
	case ActivationGeGLU:
		return &GeGLU{}
	case ActivationSiLU:
		return &SiLU{}
	case ActivationTanh:
		return &Tanh{}
	case ActivationSigmoid:
		return &Sigmoid{}
	default:
		return &GELU{} // Default to GELU
	}
}

// GELU activation function
type GELU struct{}

func (g *GELU) Forward(input *Tensor) *Tensor {
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		data[i] = 0.5 * x * (1.0 + math.Tanh(math.Sqrt(2.0/math.Pi)*(x+0.044715*math.Pow(x, 3.0))))
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (g *GELU) Backward(input, grad *Tensor) *Tensor {
	// Simplified gradient computation
	return grad
}

func (g *GELU) Name() string { return "gelu" }

// ReLU activation function
type ReLU struct{}

func (r *ReLU) Forward(input *Tensor) *Tensor {
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		if x > 0 {
			data[i] = x
		} else {
			data[i] = 0
		}
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (r *ReLU) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (r *ReLU) Name() string { return "relu" }

// SwiGLU activation function
type SwiGLU struct{}

func (s *SwiGLU) Forward(input *Tensor) *Tensor {
	// SwiGLU(x) = Swish(x) * x
	// Swish(x) = x * sigmoid(x)
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		sigmoid := 1.0 / (1.0 + math.Exp(-x))
		swish := x * sigmoid
		data[i] = swish * x
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (s *SwiGLU) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (s *SwiGLU) Name() string { return "swiglu" }

// GeGLU activation function
type GeGLU struct{}

func (g *GeGLU) Forward(input *Tensor) *Tensor {
	// GeGLU(x) = GELU(x) * x
	gelu := &GELU{}
	activated := gelu.Forward(input)
	return t.mul(activated, input)
}

func (g *GeGLU) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (g *GeGLU) Name() string { return "geglu" }

// SiLU activation function
type SiLU struct{}

func (s *SiLU) Forward(input *Tensor) *Tensor {
	// SiLU(x) = x * sigmoid(x)
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		sigmoid := 1.0 / (1.0 + math.Exp(-x))
		data[i] = x * sigmoid
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (s *SiLU) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (s *SiLU) Name() string { return "silu" }

// Tanh activation function
type Tanh struct{}

func (t *Tanh) Forward(input *Tensor) *Tensor {
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		data[i] = math.Tanh(x)
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (t *Tanh) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (t *Tanh) Name() string { return "tanh" }

// Sigmoid activation function
type Sigmoid struct{}

func (s *Sigmoid) Forward(input *Tensor) *Tensor {
	data := make([]float64, len(input.Data))
	for i, x := range input.Data {
		data[i] = 1.0 / (1.0 + math.Exp(-x))
	}
	return &Tensor{Data: data, Shape: input.Shape}
}

func (s *Sigmoid) Backward(input, grad *Tensor) *Tensor {
	return grad
}

func (s *Sigmoid) Name() string { return "sigmoid" }