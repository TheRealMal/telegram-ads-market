package model

type AuthUserRequest struct {
	Referrer int64 `json:"referrer"`
}

type SetWalletRequest struct {
	WalletAddress string `json:"wallet_address"`
}
