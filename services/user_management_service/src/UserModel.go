package usermgmt

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// User defines the user data model.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"` // Stored as a hashed password.
	Role     string `json:"role"`
}

// Validate checks that the essential fields are provided.
func (u *User) Validate() error {
	if u.Email == "" || u.Password == "" {
		return errors.New("email and password are required")
	}
	return nil
}

// HashPassword hashes the user's password using bcrypt.
func (u *User) HashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), 14)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

// CheckPassword verifies a plain-text password against the hashed version.
func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}
