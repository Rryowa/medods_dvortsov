package service

import "github.com/rryowa/medods_dvortsov/internal/storage"

type AuthService struct {
	storage storage.Storage
}

func NewAuthService(storage storage.Storage) *AuthService {
	return &AuthService{storage: storage}
}
