package main

import (
	"alice-skill/internal/logger"
	"net/http"

	"go.uber.org/zap"
)

func main() {
	parseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

// для инициализации зависимостей сервера перед запуском
func run() error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", flagRunAddr))

	// оборачиваем хендлер webhook в middleware с логированием
	return http.ListenAndServe(flagRunAddr, logger.RequestLogger(webhook))
}

// обработчик HTTP-запроса
func webhook(w http.ResponseWriter, r *http.Request) {
	// разрешён только POST-метод
	if r.Method != http.MethodPost {
		logger.Log.Debug("got request with bad method", zap.String("method", r.Method))
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

	logger.Log.Debug("sending HTTP 200 response")
}
