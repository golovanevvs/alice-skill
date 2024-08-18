package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestWebhook(t *testing.T) {
	// тип http.HandlerFunc реализует интерфейс http.Handler
	// это поможет передать хендлер тестовому серверу
	handler := http.HandlerFunc(webhook)

	// запускаем тестовый сервер
	// будет выбран первый свободный порт
	srv := httptest.NewServer(handler)

	// останавливаем сервер после завершения теста
	defer srv.Close()

	// Описываем ожидаемое тело ответа при успешном запросе
	successBody := `{
		"response": {
			"text": "Извините, я пока ничего не умею"
		},
		"version": "1.0"
}`

	// Описываем набор данных: метод запроса, ожидаемый код ответа, ожидаемое тело
	testCases := []struct {
		method       string
		expectedCode int
		expectedBody string
	}{
		{method: http.MethodGet, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{method: http.MethodPut, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{method: http.MethodDelete, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{method: http.MethodPost, expectedCode: http.StatusOK, expectedBody: successBody},
	}

	// запускаем подтесты в соответствии с testCases
	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			// делаем запрос с помощью библиотеки resty к адресу запущенного сервера, который хранится в поле URL соответствующей структуры:
			req := resty.New().R() // 1. Получаем переменную типа *resty.Request
			req.Method = tc.method // 2. Указываем метод
			req.URL = srv.URL      // 3. Указывает URL

			resp, err := req.Send() // 4. Отправляем запрос. Ответ  получаем в переменной типа *resty.Response. Не забываем обработать ошибку

			// проверяем, что нет ошибки
			assert.NoError(t, err, "error making HTTP request")

			// проверяем соответствие кода
			assert.Equal(t, tc.expectedCode, resp.StatusCode(), "Response code didn't match expected")

			// проверяем корректность полученного тела ответа, если мы его ожидаем
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, string(resp.Body()))
			}

			/* Старая версия до resty

			r := httptest.NewRequest(tc.method, "/", nil)
			w := httptest.NewRecorder()

			// Вызовем хендлер как обычную функцию, без запуска самого сервера
			webhook(w, r)

			*/
		})
	}
}
