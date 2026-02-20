package escrow

import "math"

const nanotonPerTON float64 = 1e9

// ComputeEscrowAmount returns the amount needed for escrow deposit (nanoton). Includes gas and commission. priceTON can be fractional.
func (s *service) ComputeEscrowAmount(priceTON float64) int64 {
	amountWithComission := int64(math.Round(priceTON * nanotonPerTON * s.comissionMultiplier))
	return amountWithComission + s.transactionGasNanoton
}

// GetAmountWithoutGasAndCommission extracts the price portion from the total escrow amount.
// Returns the price in TON.
func (s *service) GetAmountWithoutGasAndCommission(amountNanoton int64) int64 {
	amountWithoutGas := amountNanoton - s.transactionGasNanoton

	amountWithoutComission := float64(amountWithoutGas) / s.comissionMultiplier
	priceNanoton := int64(math.Round(amountWithoutComission))

	return priceNanoton
}
