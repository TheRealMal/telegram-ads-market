package escrow

import "math"

const nanotonPerTON float64 = 1e9

// ComputeEscrowAmount returns the amount needed for escrow deposit (nanoton). Includes gas and commission. priceNanoton is price in nanoton.
func (s *service) ComputeEscrowAmount(priceNanoton int64) int64 {
	amountWithCommission := int64(math.Round(float64(priceNanoton) * s.commissionMultiplier))
	return amountWithCommission + s.transactionGasNanoton
}

// GetAmountWithoutGasAndCommission extracts the price portion from the total escrow amount.
// Returns the price in nanoton.
func (s *service) GetAmountWithoutGasAndCommission(amountNanoton int64) int64 {
	amountWithoutGas := amountNanoton - s.transactionGasNanoton

	amountWithoutCommission := float64(amountWithoutGas) / s.commissionMultiplier
	priceNanoton := int64(math.Round(amountWithoutCommission))

	return priceNanoton
}
