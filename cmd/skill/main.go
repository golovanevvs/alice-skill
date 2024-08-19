package main

import (
	"fmt"
	"net/http"
)

func main() {
	parseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

// для инициализации зависимостей сервера перед запуском
func run() error {
	fmt.Println("Runnig server on", flagRunAddr)
	return http.ListenAndServe(flagRunAddr, http.HandlerFunc(webhook))
}

// обработчик HTTP-запроса
func webhook(w http.ResponseWriter, r *http.Request) {
	// разрешён только POST-метод
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// установка правильного заголовка для типа данных
	w.Header().Set("Content-Type", "application/json")

	// ответ-заглушка без проверки ошибок
	w.Write([]byte(`
	{
		"response": {
			"text": "Извините, я пока ничего не умею"
		},
		"version": "1.0"
	}
	`))
}
