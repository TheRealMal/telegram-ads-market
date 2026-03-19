package entity

const (
	streamKeyCryptoPayment = "events:crypto_payment"
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
	e.Address = stringFromMap(m, "address")
	e.Currency = stringFromMap(m, "currency")
	e.Amount = int64FromMap(m, "amount")
	e.TxHash = stringFromMap(m, "tx_hash")
	e.Timestamp = int64FromMap(m, "timestamp")
}

func (e *EventCryptoPayment) StreamKey() string {
	return streamKeyCryptoPayment
}
