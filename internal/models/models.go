package models

const (
	TypeSimpleUtterance = "SimpleUtterance"
)

// Описывает команду, полученную в запросе типа
type SimpleUtterance struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// Описывает запрос пользователя
// https://yandex.ru/dev/dialogs/alice/doc/request.html
type Request struct {
	Timezone string          `json:"timezone"`
	Request  SimpleUtterance `json:"request"`
	Session  Session         `json:"session"`
	Version  string          `json:"version"`
}
type Session struct {
	New bool `json:"new"`
}

// Описывает ответ, который нужно озвучить
type ResponsePayload struct {
	Text string `json:"text"`
}

// Описывает ответ сервера
// https://yandex.ru/dev/dialogs/alice/doc/response.html
type Response struct {
	Response ResponsePayload `json:"response"`
	Version  string          `json:"version"`
}
