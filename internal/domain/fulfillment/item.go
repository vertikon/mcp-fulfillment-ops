package fulfillment

// Item representa uma linha de produto em qualquer operação
type Item struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Batch    string `json:"batch,omitempty"`    // Opcional na entrada, obrigatório na saída se controlado
	Location string `json:"location,omitempty"` // Localização física (opcional)
}
