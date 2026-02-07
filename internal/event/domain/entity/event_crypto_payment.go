package entity

const (
	streamCryptoPayment = "events:crypto_payment"
)

type EventCryptoPayment struct {
	ID        string `json:"-"`
	Address   string `json:"address"`
	Currency  string `json:"currency"`
	Amount    int64  `json:"amount"`
	TxHash    string `json:"tx_hash"`
	Timestamp int64  `json:"timestamp"`
}

var _ Event = (*EventCryptoPayment)(nil)

func (e *EventCryptoPayment) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"address":   e.Address,
		"currency":  e.Currency,
		"amount":    e.Amount,
		"tx_hash":   e.TxHash,
		"timestamp": e.Timestamp,
	}
}

func (e *EventCryptoPayment) FromMap(m map[string]interface{}) {
	e.Address = m["address"].(string)
	e.Currency = m["currency"].(string)
	e.Amount = mustParseInt64(m["amount"])
	e.TxHash = m["tx_hash"].(string)
	e.Timestamp = mustParseInt64(m["timestamp"])
}

func (e *EventCryptoPayment) StreamKey() string {
	return streamCryptoPayment
}
