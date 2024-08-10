package main

import "net/http"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// для инициализации зависимостей сервера перед запуском
func run() error {
	return http.ListenAndServe(":8080", http.HandlerFunc(webhook))
}

// обработчик HTTP-запроса
func webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// установка правильного заголовка для типа данных
	w.Header().Set("Content-Type", "application/json")

	// ответ-заглушка без проверки ошибок
	_, _ = w.Write([]byte(`
	{
		"response": {
			"text": "Извините, я пока ничего не умею"
		},
		"version": "1.0"
	}
	`))
}
