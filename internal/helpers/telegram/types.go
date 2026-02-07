package telegram

const (
	CurrencyStars = "XTR"
)

// Update represents an incoming update from Telegram.
// At most one of the optional parameters can be present in any given update.
type Update struct {
	UpdateID         int64             `json:"update_id"`
	Message          *UpdateMessage    `json:"message,omitempty"`
	CallbackQuery    *CallbackQuery    `json:"callback_query,omitempty"`
	PreCheckoutQuery *PreCheckoutQuery `json:"pre_checkout_query,omitempty"`
}

// Message represents a message from Telegram.
type UpdateMessage struct {
	MessageID         int64              `json:"message_id"`
	From              *User              `json:"from,omitempty"`
	Chat              *Chat              `json:"chat"`
	Text              string             `json:"text,omitempty"`
	ReplyToMessage    *ReplyToMessage    `json:"reply_to_message,omitempty"`
	SuccessfulPayment *SuccessfulPayment `json:"successful_payment,omitempty"` // Optional. Message is a service message about a successful payment, information about the payment.
	RefundedPayment   *RefundedPayment   `json:"refunded_payment,omitempty"`
}

// ReplyToMessage is the message that this message is replying to (minimal fields for lookup).
type ReplyToMessage struct {
	MessageID int64 `json:"message_id"`
	Chat      *Chat `json:"chat"`
}

// CallbackQuery represents an incoming callback query from a callback button in an inline keyboard.
type CallbackQuery struct {
	ID      string         `json:"id"`
	From    *User          `json:"from"`
	Message *UpdateMessage `json:"message,omitempty"`
	Data    string         `json:"data,omitempty"`
}

// PreCheckoutQuery represents an incoming pre-checkout query.
type PreCheckoutQuery struct {
	ID             string `json:"id"`              // Unique query identifier
	From           *User  `json:"from"`            // User who sent the query
	Currency       string `json:"currency"`        // Three-letter ISO 4217 currency code, or “XTR” for payments in Telegram Stars.
	TotalAmount    int64  `json:"total_amount"`    // Total price in the smallest units of the currency (integer, not float/double).
	InvoicePayload string `json:"invoice_payload"` // Bot-specified invoice payload
}

// User represents a Telegram user or bot.
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Language  string `json:"language_code,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
	IsPremium bool   `json:"is_premium,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
}

// Chat represents a chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type Payment struct {
	Currency                string `json:"currency"`                   // Three-letter ISO 4217 currency code, or “XTR” for payments in Telegram Stars
	TotalAmount             int64  `json:"total_amount"`               // Total price in the smallest units of the currency (integer, not float/double).
	InvoicePayload          string `json:"invoice_payload"`            // Bot-specified invoice payload
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"` // Telegram payment identifier
}

type SuccessfulPayment struct {
	Payment
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"` // Provider payment identifier
}

type RefundedPayment struct {
	Payment
	ProviderPaymentChargeID string `json:"provider_payment_charge_id,omitempty"`
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	Height       int    `json:"height,omitempty"`
	Width        int    `json:"width,omitempty"`
}

type File struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Request payloads
// ---------------------------------------------------------------------------------------------------------------------

type ParseMode string

var (
	ModeMarkdownV2 ParseMode = "MarkdownV2"
	ModeHTML       ParseMode = "HTML"
)

type WebAppInfo struct {
	URL string `json:"url"` // An HTTPS URL of a Web App to be opened with additional data as specified in Initializing Web Apps
}

type InlineKeyboardButton struct {
	Text         string     `json:"text"`                    // Label text on the button.
	WebApp       WebAppInfo `json:"web_app,omitempty"`       // Optional. Description of the Web App that will be launched when the user presses the button.
	URL          string     `json:"url,omitempty"`           // Optional. HTTP or tg:// URL to be opened when the button is pressed.
	CallbackData string     `json:"callback_data,omitempty"` // Optional. Data to be sent in a callback query to the bot when the button is pressed, 1-64 bytes
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard,omitempty"`
}

type KeyboardButtonRequestChat struct {
	RequestID       int64 `json:"request_id"`
	ChatIsChannel   bool  `json:"chat_is_channel,omitempty"`
	ChatHasUsername bool  `json:"chat_has_username,omitempty"`
	RequestTitle    bool  `json:"request_title,omitempty"`
	RequestUsername bool  `json:"request_username,omitempty"`
	RequestPhoto    bool  `json:"request_photo,omitempty"`
}

type KeyboardButton struct {
	Text        string                    `json:"text"`
	RequestChat KeyboardButtonRequestChat `json:"request_chat,omitempty"`
}

type ReplyKeyboardMarkup struct {
	Keyboard              [][]KeyboardButton `json:"keyboard,omitempty"`
	ResizeKeyboard        bool               `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard       bool               `json:"one_time_keyboard,omitempty"`
	InputFieldPlaceholder string             `json:"input_field_placeholder,omitempty"`
}

type ReplyMarkup struct {
	ReplyKeyboardMarkup
	InlineKeyboardMarkup
}

type Payload struct {
	ChatID      int64       `json:"chat_id"`
	ParseMode   ParseMode   `json:"parse_mode"`
	ReplyMarkup ReplyMarkup `json:"reply_markup,omitempty"`
}

type PhotoPayload struct {
	Payload
	Caption string `json:"caption"`
	Photo   string `json:"photo"`
}

type MessagePayload struct {
	Payload
	Text string `json:"text"`
}

type VideoPayload struct {
	Payload
	Video   string `json:"video"`
	Caption string `json:"caption"`
}

type PreCheckoutAnswerPayload struct {
	PreCheckoutQueryID string `json:"pre_checkout_query_id"`   // Unique identifier for the query to be answered.
	OK                 bool   `json:"ok"`                      // Specify True if everything is alright (goods are available, etc.) and the bot is ready to proceed with the order. Use False if there are any problems.
	ErrorMessage       string `json:"error_message,omitempty"` // Required if ok is False. Error message in human readable form that explains the reason for failure to proceed with the checkout. Telegram will display this message to the user.
}

type LabeledPrice struct {
	Label  string `json:"label"`  // Portion label
	Amount int64  `json:"amount"` // Price of the product in the smallest units of the currency. For example, for a price of US$ 1.45 pass amount = 145.
}

type InvoiceLinkPayload struct {
	Title       string         `json:"title"`               // Product name, 1-32 characters
	Description string         `json:"description"`         // Product description, 1-255 characters
	Payload     string         `json:"payload"`             // Bot-defined invoice payload, 1-128 bytes. This will not be displayed to the user, used for internal processes.
	Currency    string         `json:"currency"`            // Three-letter ISO 4217 currency code, see more on currencies. Must be “XTR” for payments in Telegram Stars.
	Prices      []LabeledPrice `json:"prices"`              // Price breakdown, a JSON-serialized list of components (e.g. product price, tax, discount, delivery cost, delivery tax, bonus, etc.). Must contain exactly one item for payments in Telegram Stars.
	PhotoURL    string         `json:"photo_url,omitempty"` // Optional. URL of the product photo for the invoice. Can be a photo of the goods or a marketing image for a service.
}

type InvoiceLinkResponse struct {
	OK     bool   `json:"ok"`
	Result string `json:"result"`
}

type RefundStarPaymentPayload struct {
	UserID                  int64  `json:"user_id"`                    // Identifier of the user whose payment will be refunded
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"` // Telegram payment identifier
}

// SentMessage is the result of sendMessage API (message_id and chat for persistence).
type SentMessage struct {
	MessageID int64 `json:"message_id"`
	Chat      *Chat `json:"chat"`
}

// SendMessageResponse is the Telegram API response for sendMessage.
type SendMessageResponse struct {
	OK     bool         `json:"ok"`
	Result *SentMessage `json:"result"`
}
