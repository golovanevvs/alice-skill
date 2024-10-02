package pg

import (
	"alice-skill/internal/store"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// Store реализует интерфейс store.Store и позволяет взаимодействовать с СУБД PostgreSQL
type Store struct {
	// Поле conn содержит объект соединения с СУБД
	conn *sql.DB
}

// NewStore возвращает нвоый экземпляр PostgreSQL-хранилища
func NewStore(conn *sql.DB) *Store {
	return &Store{conn: conn}
}

// Bootstrap подготавливает БД к работе, создавая необходимые таблицы и индексы
func (s Store) Bootstrap(ctx context.Context) error {
	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("Ошибка запуска транзакции: %v", err)
		return err
	}

	// в случае неуспешного коммита все изменения транзации будут отменены
	defer tx.Rollback()

	// создаём таблицу пользователей и необходимые индексы
	tx.ExecContext(ctx, `
	CREATE TABLE users (
		id VARCHAR(128) PRIMARY KEY,
		username VARCHAR(128)
	);
	`)

	tx.ExecContext(ctx, `
	CREATE UNIQUE INDEX sender_idx ON users (username);
	`)

	// создаём таблицу сообщений и необходимые индексы
	tx.ExecContext(ctx, `
	CREATE TABLE messages (
		id SERIAL PRIMARY KEY,
		sender VARCHAR(128),
		recipient VARCHAR(128),
		payload TEXT,
		sent_at TIMESTAMP WITH TIME ZONE,
		read_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
	);
	`)

	tx.ExecContext(ctx, `
	CREATE INDEX recipient_idx ON messages (recipient);
	`)

	// коммитим транзакцию
	return tx.Commit()
}

// FindRecipient ищет в БД userID по username
func (s Store) FindRecipient(ctx context.Context, username string) (userID string, err error) {
	// запрашиваем внутренний идентификатор пользователя по его имени
	row := s.conn.QueryRowContext(ctx, `
	SELECT id FROM users
	WHERE username = $1
	`, username)

	err = row.Scan(&userID)
	if err != nil {
		fmt.Printf("Ошибка запроса внутреннего идентификатора пользователя по его имени: %v", err)
	}

	return
}

// ListMessages ищет в БД все сообщения пользователя с userID
func (s Store) ListMessages(ctx context.Context, userID string) ([]store.Message, error) {
	// запрашиваем данные обо всех сообщениях пользователя, без самого текста
	rows, err := s.conn.QueryContext(ctx, `
	SELECT
		m.id,
		u.username AS sender,
		m.sent_at
	FROM messages m
	JOIN users u ON m.sender = u.id
	WHERE m.recipient = $1;
	`, userID)

	if err != nil {
		fmt.Printf("Ошибка получения сообщений из БД: %v", err)
		return nil, err
	}

	// не забываем закрыть курсор после завершения работы с данными
	defer rows.Close()

	// считываем записи в слайс сообщений
	var messages []store.Message

	for rows.Next() {
		var m store.Message
		if err := rows.Scan(&m.ID, &m.Sender, &m.Time); err != nil {
			fmt.Printf("Ошибка считывания записи из БД в слайс: %v", err)
			return nil, err
		}
		messages = append(messages, m)
	}

	// проверка ошибки уровня курсора
	if err := rows.Err(); err != nil {
		fmt.Printf("Ошибка уровня курсора: %v", err)
		return nil, err
	}

	return messages, nil
}

// GetMessage получает сообщение по внутреннему идентификатору
func (s Store) GetMessage(ctx context.Context, id int64) (*store.Message, error) {
	// запрашиваем сообщение по внутреннему идентификатору
	row := s.conn.QueryRowContext(ctx, `
	SELECT
		m.id,
		u.username AS sender,
		m.payload,
		m.sent_at
	FROM messages m
	JOIN users u ON m.sender = u.id
	WHERE m.id = $1;
	`, id)

	// считываем значения из записи БД в соответствующие поля структуры
	var msg store.Message
	err := row.Scan(&msg.ID, &msg.Sender, &msg.Payload, &msg.Time)
	if err != nil {
		fmt.Printf("Ошибка получения данных из БД: %v", err)
		return nil, err
	}
	return &msg, nil
}

// SaveMessage добавляет новое сообщение в БД
func (s Store) SaveMessages(ctx context.Context, messages ...store.Message) error {
	// соберём данные для создания запроса с групповой вставкой
	var values []string
	var args []any
	for i, msg := range messages {
		// в нашем запросе по 4 параметра на каждое сообщение
		base := i * 4
		// PostgreSQL требует шаблоны в формате ($1, $2, $3, $4) для каждой вставки
		params := fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
		values = append(values, params)
		args = append(args, msg.Sender, msg.Recepient, msg.Payload, msg.Time)
	}

	// составляем строку запроса
	query := `
		INSERT INTO messages
			(sender, recepient, payload, sent_at)
		VALUES 
	` + strings.Join(values, ",") + `;`

	// добавляем новые сообщения в БД
	_, err := s.conn.ExecContext(ctx, query, args...)

	if err != nil {
		fmt.Printf("Ошибка добавления сообщения в БД: %v", err)
	}

	return err
}

// RegisterUser добавляет новую запись пользователя
func (s Store) RegisterUser(ctx context.Context, userID, username string) error {
	// добавляем новую запись пользователя
	_, err := s.conn.ExecContext(ctx, `
		INSERT INTO users
			(id, username)
		VALUES
			($1, $2);
		`, userID, username)
	if err != nil {
		// проверяем, что ошибка сигнализирует о потенциальном нарушении целостности данных
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			err = store.ErrConflict
		}
	}
	return err
}
