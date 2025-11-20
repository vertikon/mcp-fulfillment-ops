package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// FulfillmentSubscriber gerencia a assinatura de eventos de entrada (OMS/WMS)
type FulfillmentSubscriber struct {
	js      jetstream.JetStream
	useCase *app.ShipOrderUseCase
	logger  Logger
}

// Logger is defined in logger_adapter.go

// NewFulfillmentSubscriber cria uma nova instância do subscriber
func NewFulfillmentSubscriber(js jetstream.JetStream, useCase *app.ShipOrderUseCase, logger Logger) *FulfillmentSubscriber {
	return &FulfillmentSubscriber{
		js:      js,
		useCase: useCase,
		logger:  logger,
	}
}

// OrderReadyToPickEvent representa o evento do OMS quando um pedido está pronto para separação
type OrderReadyToPickEvent struct {
	OrderID     string `json:"order_id"`
	Customer    string `json:"customer_name"`
	Destination string `json:"shipping_address"`
	Priority    int    `json:"priority"`
	Items       []struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	} `json:"items"`
	Metadata struct {
		TraceID string `json:"trace_id"`
		Source  string `json:"source"`
	} `json:"metadata"`
}

// Start inicia o consumo das mensagens do NATS
func (s *FulfillmentSubscriber) Start(ctx context.Context) error {
	// Criar ou atualizar stream se necessário
	streamName := "OMS_EVENTS"
	_, err := s.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"oms.order.ready_to_pick.v1"},
	})
	if err != nil {
		// Stream pode já existir, continuar
		s.logger.Info("Stream may already exist", zap.String("stream", streamName))
	}

	// Criar ou atualizar consumer
	consumer, err := s.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       "fulfillment-ops-worker",
		Description:   "Processa pedidos prontos para separação",
		FilterSubject: "oms.order.ready_to_pick.v1",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	s.logger.Info("Listening for 'oms.order.ready_to_pick.v1' events...")

	// Consumir mensagens
	iter, err := consumer.Messages()
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	// Processar mensagens em goroutine
	go s.processMessages(ctx, iter)

	return nil
}

func (s *FulfillmentSubscriber) processMessages(ctx context.Context, iter jetstream.MessagesContext) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Subscriber context cancelled, stopping...")
			return
		default:
			msg, err := iter.Next()
			if err != nil {
				s.logger.Error("Failed to get next message", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}

			// Processar mensagem em goroutine separada
			go func(m jetstream.Msg) {
				if err := s.handleOrderEvent(ctx, m); err != nil {
					s.logger.Error("Failed to process order event", zap.Error(err))
					m.Nak() // Nack para retentativa
					return
				}
				m.Ack() // Sucesso
			}(msg)
		}
	}
}

// handleOrderEvent processa um evento de pedido pronto para separação
func (s *FulfillmentSubscriber) handleOrderEvent(ctx context.Context, msg jetstream.Msg) error {
	var event OrderReadyToPickEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("invalid json format: %w", err)
	}

	s.logger.Info("Receiving Order", zap.String("order_id", event.OrderID))

	// Mapear para domínio
	domainItems := make([]fulfillment.Item, len(event.Items))
	for i, item := range event.Items {
		domainItems[i] = fulfillment.Item{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		}
	}

	// Criar FulfillmentOrder via caso de uso
	_, err := s.useCase.CreateOrder(ctx, event.OrderID, event.Customer, event.Destination, domainItems, event.Priority)
	if err != nil {
		return fmt.Errorf("failed to create fulfillment order: %w", err)
	}

	return nil
}

