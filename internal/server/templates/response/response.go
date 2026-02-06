// pkg/response/response.go
package response

// Template - стандартный шаблон ответа API
// swagger:model Template
type Template struct {
	Ok        bool        `json:"ok"`
	Data      interface{} `json:"data,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
}
