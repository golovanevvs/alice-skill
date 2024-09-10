package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Описываем набор данных: метод запроса, ожидаемый код ответа, ожидаемое тело
	testCases := []struct {
		name         string
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "method_get",
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "method_put",
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "method_delete",
			method:       http.MethodDelete,
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "",
		},
		{
			name:         "method_post_without_body",
			method:       http.MethodPost,
			expectedCode: http.StatusInternalServerError,
			expectedBody: "",
		},
		{
			name:         "method_post_unsupported_type",
			method:       http.MethodPost,
			body:         `{"request": {"type": "idunno", "command": "do something"}, "version": "1.0"}`,
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: "",
		},
		{
			name:         "method_post_success",
			method:       http.MethodPost,
			body:         `{"request": {"type": "SimpleUtterance", "command": "sudo do something"}, "version": "1.0"}`,
			expectedCode: http.StatusOK,
			// ответ стал сложнее, поэтому сравниваем его с шаблоном вместо точной строки
			expectedBody: `Точное время .* часов, .* минут. Для вас нет новых сообщений.`,
		},
	}

	// запускаем подтесты в соответствии с testCases
	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			// делаем запрос с помощью библиотеки resty к адресу запущенного сервера, который хранится в поле URL соответствующей структуры:
			req := resty.New().R() // 1. Получаем переменную типа *resty.Request
			req.Method = tc.method // 2. Указываем метод
			req.URL = srv.URL      // 3. Указывает URL

			if len(tc.body) > 0 {
				req.SetHeader("Content-Type", "application/json")
				req.SetBody(tc.body)
			}

			resp, err := req.Send() // 4. Отправляем запрос. Ответ  получаем в переменной типа *resty.Response. Не забываем обработать ошибку

			// проверяем, что нет ошибки
			assert.NoError(t, err, "error making HTTP request")

			// проверяем соответствие кода
			assert.Equal(t, tc.expectedCode, resp.StatusCode(), "Response code didn't match expected")

			// проверяем корректность полученного тела ответа, если мы его ожидаем
			// сравниваем тело ответа с ожидаемым шаблоном
			if tc.expectedBody != "" {
				assert.Regexp(t, tc.expectedBody, string(resp.Body()))
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

func TestGzipCompression(t *testing.T) {
	handler := http.HandlerFunc(gzipMiddleware(webhook))

	srv := httptest.NewServer(handler)
	defer srv.Close()

	requestBody := `{
		"request": {
			"type": "SimpleUtterance",
			"command": "sudo do something"
		},
		"version": "1.0"
	}`

	// ожидаемое содержимое тела ответа при успешном запросе
	successBody := `{
		"response": {
			"text": "Извините, я пока ничего не умею"
		},
		"version": "1.0"
	}`

	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest("POST", srv.URL, buf)
		r.RequestURI = ""
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.JSONEq(t, successBody, string(b))
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest("POST", srv.URL, buf)
		r.RequestURI = ""
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		b, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.JSONEq(t, successBody, string(b))
	})
}
