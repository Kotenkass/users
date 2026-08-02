package db

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"

	dbhandler "github.com/KOTENKASS/users/lib"
)

// User is the PostgreSQL users table model.
type User struct {
	tableName struct{} `pg:"users"`

	ID           int       `pg:"id,pk"`
	ChatID       int64     `pg:"chat_id,unique,notnull"`
	TelegramID   int64     `pg:"telegram_id,unique,notnull"`
	FirstName    string    `pg:"first_name"`
	LastName     string    `pg:"last_name"`
	LanguageCode string    `pg:"language_code"`
	Username     string    `pg:"username"`
	State        string    `pg:"state"`
	CreationTime time.Time `pg:"creation_time,default:now()"`
}

// UsersDBHandler embeds the shared DBHandler and adds users-specific operations.
type UsersDBHandler struct {
	dbhandler.DBHandler
}

// CreateUser inserts a new user and returns the created row.
func (db *UsersDBHandler) CreateUser(chatID, telegramID int64, firstName, lastName, languageCode, username string) (*User, error) {
	user := &User{
		ChatID:       chatID,
		TelegramID:   telegramID,
		FirstName:    firstName,
		LastName:     lastName,
		LanguageCode: languageCode,
		Username:     username,
	}

	if _, err := db.DBHandler.Conn.Model(user).Insert(); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser returns a user by chat_id.
func (db *UsersDBHandler) GetUser(chatID int64) (*User, error) {
	user := new(User)
	err := db.DBHandler.Conn.Model(user).Where("chat_id = ?", chatID).First()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser updates mutable Telegram profile fields by chat_id.
func (db *UsersDBHandler) UpdateUser(chatID int64, firstName, lastName, languageCode, username string) (*User, error) {
	_, err := db.DBHandler.Conn.Model((*User)(nil)).
		Set("first_name = ?", firstName).
		Set("last_name = ?", lastName).
		Set("language_code = ?", languageCode).
		Set("username = ?", username).
		Where("chat_id = ?", chatID).
		Update()
	if err != nil {
		return nil, err
	}

	return db.GetUser(chatID)
}

// ListUsers returns all users ordered by creation time.
func (db *UsersDBHandler) ListUsers() ([]User, error) {
	var users []User
	err := db.DBHandler.Conn.Model(&users).OrderExpr("creation_time ASC, id ASC").Select()
	if err != nil {
		return nil, err
	}

	return users, nil
}

// DeleteUser deletes a user by chat_id.
func (db *UsersDBHandler) DeleteUser(chatID int64) error {
	_, err := db.DBHandler.Conn.Model((*User)(nil)).Where("chat_id = ?", chatID).Delete()
	if errors.Is(err, pg.ErrNoRows) {
		return nil
	}
	return err
}
