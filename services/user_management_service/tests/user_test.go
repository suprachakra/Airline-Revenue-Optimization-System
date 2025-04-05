package usermgmt_test

import (
	"testing"
	"iaros/user_management_service"
	"github.com/stretchr/testify/assert"
)

func TestUserRegistrationAndHashing(t *testing.T) {
	user := &usermgmt.User{
		ID:       "user123",
		Email:    "test@example.com",
		Password: "securepassword",
		Role:     "user",
	}
	err := user.Validate()
	assert.NoError(t, err, "User validation should succeed")
	err = user.HashPassword()
	assert.NoError(t, err, "Password hashing should complete without error")
	assert.NotEqual(t, "securepassword", user.Password, "Password should be hashed")
}

func TestPasswordVerification(t *testing.T) {
	user := &usermgmt.User{
		ID:       "user123",
		Email:    "test@example.com",
		Password: "securepassword",
		Role:     "user",
	}
	_ = user.HashPassword()
	assert.True(t, user.CheckPassword("securepassword"), "Password verification should succeed")
	assert.False(t, user.CheckPassword("wrongpassword"), "Password verification should fail for incorrect password")
}
