package entity

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Photo     string `json:"photo"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Locale    string `json:"locale"`
}
