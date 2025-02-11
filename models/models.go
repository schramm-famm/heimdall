package models

import "github.com/dgrijalva/jwt-go"

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenClaims struct {
	User
	jwt.StandardClaims
}

type AuthBody struct {
	Token string `json:"token"`
}
