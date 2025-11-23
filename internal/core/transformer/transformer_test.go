// Package transformer provides transformer tests
package transformer

import (
	"context"
	"testing"
)

func TestNewGLMTransformer(t *testing.T) {
	config := GLMConfig{
		VocabSize:  10000,
		HiddenSize: 512,
		NumLayers:  6,
		NumHeads:   8,
		MaxSeqLen:  2048,
		Dropout:    0.1,
		Attention: AttentionConfig{
			NumHeads: 8,
			HeadDim:  64,
			Dropout:  0.1,
		},
		FeedForward: FeedForwardConfig{
			HiddenSize:       512,
			IntermediateSize: 2048,
			Dropout:          0.1,
			Activation:       "gelu",
		},
		LayerNormEps: 1e-5,
	}

	transformer := NewGLMTransformer(config)
	if transformer == nil {
		t.Fatal("NewGLMTransformer() returned nil")
	}

	if len(transformer.layers) != config.NumLayers {
		t.Errorf("NewGLMTransformer() layers = %v, want %v", len(transformer.layers), config.NumLayers)
	}

	if transformer.embeddings == nil {
		t.Error("NewGLMTransformer() embeddings is nil")
	}

	if transformer.posEncoding == nil {
		t.Error("NewGLMTransformer() posEncoding is nil")
	}
}

func TestGLMTransformer_Forward(t *testing.T) {
	config := GLMConfig{
		VocabSize:  10000,
		HiddenSize: 512,
		NumLayers:  2,
		NumHeads:   8,
		MaxSeqLen:  1024,
		Dropout:    0.1,
		Attention: AttentionConfig{
			NumHeads: 8,
			HeadDim:  64,
		},
		FeedForward: FeedForwardConfig{
			HiddenSize:       512,
			IntermediateSize: 2048,
			Activation:       "gelu",
		},
		LayerNormEps: 1e-5,
	}

	transformer := NewGLMTransformer(config)

	input := &Tensor{
		Data:  make([]float64, 10*512),
		Shape: []int{10, 512},
	}

	ctx := context.Background()
	output, err := transformer.Forward(ctx, input, nil)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}

	if output == nil {
		t.Fatal("Forward() returned nil output")
	}
}
