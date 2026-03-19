package entity

const streamKeyEscrowDeposit = "events:escrow_deposit"

type EventEscrowDeposit struct {
	ID        string `json:"-"`
	Address   string `json:"address"`   // raw TON address (same as Redis key)
	Amount    int64  `json:"amount"`    // nanoton
	Timestamp int64  `json:"timestamp"` // unix
	TxHash    string `json:"tx_hash"`
}

var _ Event = (*EventEscrowDeposit)(nil)

func (e *EventEscrowDeposit) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"address":   e.Address,
		"amount":    e.Amount,
		"timestamp": e.Timestamp,
		"tx_hash":   e.TxHash,
	}
}

func (e *EventEscrowDeposit) FromMap(m map[string]interface{}) {
	e.Address = stringFromMap(m, "address")
	e.Amount = int64FromMap(m, "amount")
	e.Timestamp = int64FromMap(m, "timestamp")
	e.TxHash = stringFromMap(m, "tx_hash")
}

func (e *EventEscrowDeposit) StreamKey() string {
	return streamKeyEscrowDeposit
}
