package bycript

import (
	"golang.org/x/crypto/bcrypt"
)

type Password interface {
	HashPassword(pass *string)
	CheckPasswordHash(phash []byte, pass string) bool
}

type password struct{}

func NewHashPassword() Password {
	return &password{}
}

func (p *password) HashPassword(pass *string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(*pass), bcrypt.DefaultCost)
	*pass = string(hash)
}

func (p *password) CheckPasswordHash(hash []byte, pass string) bool {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(pass)); err == nil {
		hash = nil
		return true
	}
	return false
}
