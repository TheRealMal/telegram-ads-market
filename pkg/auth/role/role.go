package role

type Role string

const (
	UserRole  Role = "user"
	AdminRole Role = "admin"
	EmptyRole Role = ""
)

func FromString(s string) Role {
	switch s {
	case "user":
		return UserRole
	case "admin":
		return AdminRole
	default:
		return EmptyRole
	}
}

func (r Role) String() string {
	return string(r)
}
