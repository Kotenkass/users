package actions

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// User is a simple DTO used by the REST API.
// In a real service this would map to a database model.
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
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

// ListUsers returns a placeholder list of users.
func ListUsers(c echo.Context) error {
	users := []User{
		{ID: 1, Name: "Alice Example", Email: "alice@example.com"},
		{ID: 2, Name: "Bob Example", Email: "bob@example.com"},
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": users,
	})
}

// CreateUser reads a request body and returns the created user.
func CreateUser(c echo.Context) error {
	var payload User
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	// Placeholder for real database create operation.
	payload.ID = 1001

	return c.JSON(http.StatusCreated, map[string]any{
		"data": payload,
	})
}

// GetUser returns a placeholder user by ID.
func GetUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	// Placeholder for real database lookup.
	user := User{
		ID:    id,
		Name:  "Alice Example",
		Email: "alice@example.com",
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": user,
	})
}

// UpdateUser replaces a placeholder user by ID.
func UpdateUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	var payload User
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	payload.ID = id

	// Placeholder for real database update operation.
	return c.JSON(http.StatusOK, map[string]any{
		"data": payload,
	})
}

// PatchUser partially updates a placeholder user by ID.
func PatchUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	// Placeholder for real database partial-update operation.
	user := User{
		ID:    id,
		Name:  "Alice Example",
		Email: "alice@example.com",
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": user,
	})
}

// DeleteUser deletes a placeholder user by ID.
func DeleteUser(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	// Placeholder for real database delete operation.
	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"deleted": true,
			"id":      id,
		},
	})
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
