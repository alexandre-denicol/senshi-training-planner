package users

import "time"

type Role string

const (
	RoleAdmin     Role = "ADMIN"
	RoleProfessor Role = "PROFESSOR"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
