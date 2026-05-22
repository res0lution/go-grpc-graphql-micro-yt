package model

import "time"

type User struct {
	ID             string    `json:"id"`
	IdentityID     string    `json:"identity_id"`
	Sub            string    `json:"sub"`
	Login          string    `json:"login"`
	Email          string    `json:"email"`
	EmployeeNumber string    `json:"employee_number"`
	WinAccountName string    `json:"winaccountname"`
	GivenName      string    `json:"given_name"`
	FamilyName     string    `json:"family_name"`
	Name           string    `json:"name"`
	// Public user payload uses plural "groups" field.
	Groups         []string  `json:"groups"`
	IsActive       bool      `json:"is_active"`
	LastLoginAt    time.Time `json:"last_login_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
