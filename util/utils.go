package util

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func ComparePassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func GenerateToken(userId string) (map[string]string, error) {
	key := []byte(os.Getenv("SECRET_KEY"))
	access_jwt_instance := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"user_id": userId,
			"type":    "access",
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
		})
	access_token, err := access_jwt_instance.SignedString(key)
	if err != nil {
		return nil, err
	}
	refresh_jwt_instance := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"user_id": userId,
			"type":    "refresh",
			"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
		})
	refresh_token, err := refresh_jwt_instance.SignedString(key)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"access_token":  access_token,
		"refresh_token": refresh_token,
	}, nil
}

func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	key := []byte(os.Getenv("SECRET_KEY"))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func GenerateOtp() (string, error) {
	otpBytes := make([]byte, 6)
	_, err := rand.Read(otpBytes)
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		otpBytes[i] = '0' + (otpBytes[i] % 10)
	}
	return string(otpBytes), nil
}

func RenderTemplate(templatePath string, data interface{}) (string, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", err
	}

	return rendered.String(), nil
}