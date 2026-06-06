package identity

import "time"

// Role mirrors the user_role PostgreSQL enum.
type Role string

const (
	RoleOwner    Role = "owner"
	RoleBarber   Role = "barber"
	RoleCustomer Role = "customer"
)

type Shop struct {
	ID        [16]byte
	Name      string
	Slug      string
	Phone     string
	Address   string
	City      string
	State     string
	Timezone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID           [16]byte
	Email        string
	PasswordHash string
	FullName     string
	Phone        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Membership struct {
	ID        [16]byte
	ShopID    [16]byte
	UserID    [16]byte
	Role      Role
	CreatedAt time.Time
}
