package nats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// InventoryCommandClient implementa o contrato InventoryClient para comunicação com mcp-core-inventory
type InventoryCommandClient struct {
	baseURL    string
	httpClient *http.Client
	logger     Logger
}

// Logger is defined in logger_adapter.go

// NewInventoryCommandClient cria uma nova instância do cliente
func NewInventoryCommandClient(baseURL string, logger Logger) *InventoryCommandClient {
	return &InventoryCommandClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// AdjustStock ajusta o estoque no Core Inventory
func (c *InventoryCommandClient) AdjustStock(ctx context.Context, location string, sku string, quantity int, batch string) error {
	reqBody := map[string]interface{}{
		"location": location,
		"sku":      sku,
		"quantity": quantity,
		"batch":    batch,
		"reason":   "fulfillment_operation",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/adjust", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call core inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("core inventory returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Stock adjusted successfully", zap.String("location", location), zap.String("sku", sku), zap.Int("quantity", quantity))
	return nil
}

// ConfirmReservation confirma uma reserva no Core Inventory
func (c *InventoryCommandClient) ConfirmReservation(ctx context.Context, orderID string, items []fulfillment.Item) error {
	reqBody := map[string]interface{}{
		"order_id": orderID,
		"items":    items,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/reserve/confirm", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call core inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("core inventory returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Reservation confirmed successfully", zap.String("order_id", orderID))
	return nil
}

// GetAvailableStock obtém o estoque disponível do Core Inventory
func (c *InventoryCommandClient) GetAvailableStock(ctx context.Context, location string, sku string) (int, error) {
	url := fmt.Sprintf("%s/v1/available?location=%s&sku=%s", c.baseURL, location, sku)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call core inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("core inventory returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Available int `json:"available"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Available, nil
}

