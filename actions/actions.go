package actions

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/KOTENKASS/users/db"
	"github.com/labstack/echo/v4"
)

// User is a DTO used by the REST API.
type User struct {
	ID           int64     `json:"id"`
	ChatID       int64     `json:"chatID"`
	TelegramID   int64     `json:"telegramID"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	LanguageCode string    `json:"languageCode"`
	Username     string    `json:"username"`
	State        string    `json:"state,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// RegisterRoutes configures all users microservice routes.
func RegisterRoutes(e *echo.Echo) {
	users := e.Group("/users")

	users.GET("", ListUsers)
	users.POST("", CreateUser)
	users.GET("/:id", GetUser)
	users.PUT("/:id", UpdateUser)
	users.PATCH("/:id", PatchUser)
	users.DELETE("/:id", DeleteUser)
}

// ListUsers returns users from the database.
func ListUsers(c echo.Context) error {
	var users []User
	err := withUsersDB(func(dbh *db.UsersDBHandler) error {
		dbUsers, err := dbh.ListUsers()
		if err != nil {
			return err
		}

		users = usersFromDB(dbUsers)
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": users,
	})
}

// CreateUser reads a request body and creates a user in the database.
func CreateUser(c echo.Context) error {
	var payload User
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var created User
	err := withUsersDB(func(dbh *db.UsersDBHandler) error {
		dbUser, err := dbh.CreateUser(payload.ChatID, payload.TelegramID, payload.FirstName, payload.LastName, payload.LanguageCode, payload.Username)
		if err != nil {
			return err
		}

		created = userFromDB(dbUser)
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": created,
	})
}

// GetUser returns a user by chat_id.
func GetUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var dbUser *db.User
	err = withUsersDB(func(dbh *db.UsersDBHandler) error {
		found, err := dbh.GetUser(id)
		if err != nil {
			return err
		}

		dbUser = found
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}
	if dbUser == nil {
		return c.JSON(http.StatusNotFound, errorResponse("user not found"))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": userFromDB(dbUser),
	})
}

// UpdateUser replaces mutable Telegram profile fields for a user.
func UpdateUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var payload User
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var updated *db.User
	err = withUsersDB(func(dbh *db.UsersDBHandler) error {
		found, err := dbh.UpdateUser(id, payload.FirstName, payload.LastName, payload.LanguageCode, payload.Username)
		if err != nil {
			return err
		}

		updated = found
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}
	if updated == nil {
		return c.JSON(http.StatusNotFound, errorResponse("user not found"))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": userFromDB(updated),
	})
}

// PatchUser partially updates mutable Telegram profile fields for a user.
func PatchUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var payload User
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var updated *db.User
	err = withUsersDB(func(dbh *db.UsersDBHandler) error {
		existing, err := dbh.GetUser(id)
		if err != nil {
			return err
		}
		if existing == nil {
			return nil
		}

		if payload.FirstName != "" {
			existing.FirstName = payload.FirstName
		}
		if payload.LastName != "" {
			existing.LastName = payload.LastName
		}
		if payload.LanguageCode != "" {
			existing.LanguageCode = payload.LanguageCode
		}
		if payload.Username != "" {
			existing.Username = payload.Username
		}

		patched, err := dbh.UpdateUser(existing.ChatID, existing.FirstName, existing.LastName, existing.LanguageCode, existing.Username)
		if err != nil {
			return err
		}

		updated = patched
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}
	if updated == nil {
		return c.JSON(http.StatusNotFound, errorResponse("user not found"))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": userFromDB(updated),
	})
}

// DeleteUser deletes a user by chat_id.
func DeleteUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	err = withUsersDB(func(dbh *db.UsersDBHandler) error {
		existing, err := dbh.GetUser(id)
		if err != nil {
			return err
		}
		if existing == nil {
			return nil
		}

		return dbh.DeleteUser(id)
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"deleted": true,
			"id":      id,
		},
	})
}

func withUsersDB(handler func(*db.UsersDBHandler) error) error {
	dbh := db.UsersDBHandler{}
	if err := dbh.ConnectPg(); err != nil {
		return err
	}
	defer dbh.Close()

	return handler(&dbh)
}

func usersFromDB(dbUsers []db.User) []User {
	users := make([]User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		users = append(users, userFromDB(&dbUser))
	}
	return users
}

func userFromDB(dbUser *db.User) User {
	if dbUser == nil {
		return User{}
	}

	return User{
		ID:           int64(dbUser.ID),
		ChatID:       dbUser.ChatID,
		TelegramID:   dbUser.TelegramID,
		FirstName:    dbUser.FirstName,
		LastName:     dbUser.LastName,
		LanguageCode: dbUser.LanguageCode,
		Username:     dbUser.Username,
		State:        dbUser.State,
		CreationTime: dbUser.CreationTime,
	}
}

func parseIDParam(c echo.Context) (int64, error) {
	rawID := c.Param("id")
	if rawID == "" {
		return 0, errors.New("missing user id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid user id")
	}

	if id <= 0 {
		return 0, errors.New("user id must be positive")
	}

	return id, nil
}

func errorResponse(message string) map[string]any {
	return map[string]any{
		"error": message,
	}
}
