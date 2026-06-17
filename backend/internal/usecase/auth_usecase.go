package usecase

import (
	"errors"
	"fmt"
	"os"
	"time"

	"fleamarket-backend/internal/domain"
	"fleamarket-backend/internal/infrastructure/persistence"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
)

type AuthUsecase struct {
	userRepo *persistence.UserRepository
}

func NewAuthUsecase(userRepo *persistence.UserRepository) *AuthUsecase {
	return &AuthUsecase{userRepo: userRepo}
}

type SignupInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	Token string      `json:"token"`
	User  *domain.User `json:"user"`
}

func (u *AuthUsecase) Signup(input SignupInput) (*AuthOutput, error) {
	existing, err := u.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("checking email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		Name:           input.Name,
		Email:          input.Email,
		HashedPassword: string(hashed),
	}
	if err := u.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &AuthOutput{Token: token, User: user}, nil
}

func (u *AuthUsecase) Login(input LoginInput) (*AuthOutput, error) {
	user, err := u.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &AuthOutput{Token: token, User: user}, nil
}

func generateJWT(user *domain.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
