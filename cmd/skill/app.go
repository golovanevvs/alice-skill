package main

import (
	"alice-skill/internal/logger"
	"alice-skill/internal/models"
	"alice-skill/internal/store"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// app инкапсулирует в себя все зависимости и логику приложения
type app struct {
	store store.Store
}

// newApp принимает на вход внешние зависимости приложения и возвращает новый объект app
func newApp(s store.Store) *app {
	return &app{store: s}
}

// обработчик HTTP-запроса
func (a *app) webhook(w http.ResponseWriter, r *http.Request) {
	var req models.Request

	ctx := r.Context()

	// разрешён только POST-метод
	if r.Method != http.MethodPost {
		logger.Log.Debug("got request with bad method", zap.String("method", r.Method))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// десериализуем запрос в структуру модели
	logger.Log.Debug("decoding request")

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// проверяем, что пришёл запрос понятного типа
	if req.Request.Type != models.TypeSimpleUtterance {
		logger.Log.Debug("unsupported request type", zap.String("type", req.Request.Type))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// получаем список сообщений для текущего пользователя
	messages, err := a.store.ListMessages(ctx, req.Session.User.UserID)
	if err != nil {
		logger.Log.Debug("cannot load messages for user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}

	// формируем текст с количеством сообщений
	text := "Для вас нет новых сообщений."
	if len(messages) > 0 {
		text = fmt.Sprintf("Для вас %d новых сообщений.", len(messages))
	}

	// первый запрос новой сессии
	if req.Session.New {
		// обрабатываем поле Timezone запроса
		tz, err := time.LoadLocation(req.Timezone)
		if err != nil {
			logger.Log.Debug("cannot parse timezone")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// получаем текущее время в часовом поясе пользователя
		now := time.Now().In(tz)
		hour, minute, _ := now.Clock()

		// формируем новый текст приветствия
		text = fmt.Sprintf("Точное время %d часов, %d минут. %s", hour, minute, text)
	}

	// заполняем модель ответа
	resp := models.Response{
		Response: models.ResponsePayload{
			Text: text,
		},
		Version: "1.0",
	}

	// установка правильного заголовка для типа данных
	w.Header().Set("Content-Type", "application/json")

	// сериализуем ответ сервера
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}

	logger.Log.Debug("sending HTTP 200 response")
}
