package persistence

import (
	"errors"

	"fleamarket-backend/internal/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) UpdateAddress(id uint, postalCode, prefecture, city, addressLine, building string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"postal_code":  postalCode,
		"prefecture":   prefecture,
		"city":         city,
		"address_line": addressLine,
		"building":     building,
	}).Error
}
