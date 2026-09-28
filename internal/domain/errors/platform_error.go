package platformError

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
