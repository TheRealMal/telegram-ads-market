package escrow

import "math"

const nanotonPerTON float64 = 1e9

// ComputeEscrowAmount returns the amount needed for escrow deposit (nanoton). Includes gas and commission. priceNanoton is price in nanoton.
func (s *service) ComputeEscrowAmount(priceNanoton int64) int64 {
	amountWithComission := int64(math.Round(float64(priceNanoton) * s.comissionMultiplier))
	return amountWithComission + s.transactionGasNanoton
}

// GetAmountWithoutGasAndCommission extracts the price portion from the total escrow amount.
// Returns the price in nanoton.
func (s *service) GetAmountWithoutGasAndCommission(amountNanoton int64) int64 {
	amountWithoutGas := amountNanoton - s.transactionGasNanoton

	amountWithoutComission := float64(amountWithoutGas) / s.comissionMultiplier
	priceNanoton := int64(math.Round(amountWithoutComission))

	return priceNanoton
}
