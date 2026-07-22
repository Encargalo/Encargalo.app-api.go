package bycript

import (
	"golang.org/x/crypto/bcrypt"
)

type Password interface {
	HashPassword(pass *string) error
	CheckPasswordHash(phash []byte, pass string) bool
}

type password struct{}

func NewHashPassword() Password {
	return &password{}
}

func (p *password) HashPassword(pass *string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(*pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	*pass = string(hash)
	return nil
}

func (p *password) CheckPasswordHash(hash []byte, pass string) bool {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(pass)); err == nil {
		return true
	}
	return false
}
