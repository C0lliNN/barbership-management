package identity

type Role string

const (
	RoleOwner    Role = "owner"
	RoleBarber   Role = "barber"
	RoleCustomer Role = "customer"
)

type Shop struct {
	ID        string
	Name      string
	Slug      string
	Phone     string
	Address   string
	City      string
	State     string
	CreatedAt int64
	UpdatedAt int64
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Phone        string
	CreatedAt    int64
	UpdatedAt    int64
}

type Membership struct {
	ID        string
	ShopID    string
	UserID    string
	Role      Role
	CreatedAt int64
}
