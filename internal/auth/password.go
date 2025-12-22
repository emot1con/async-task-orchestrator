package auth

import "golang.org/x/crypto/bcrypt"

func GeneratePasswordHash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func ComparePasswordHash(hashedPassword []byte, password string) error {
	// Compare password with hash
	return bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
}
