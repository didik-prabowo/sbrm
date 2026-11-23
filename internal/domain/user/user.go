package user

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         Role
}

type Principal struct {
	UserID int64
	Role   Role
}

func (p Principal) IsAdmin() bool { return p.Role == RoleAdmin }
