package services

import (
	"context"
	"errors"
	"net/http"
	"time"
	"url-shortener/internals/dtos"
	"url-shortener/internals/models"
	"url-shortener/internals/repository"
	"url-shortener/internals/utils"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo   *repository.AuthRepo
	logger *zap.Logger
}

func NewAuthService(db *mongo.Database, logger *zap.Logger) *AuthService {
	return &AuthService{
		repo:   repository.NewAuthRepo(db, logger),
		logger: logger,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, registerUserDto dtos.RegisterUserDto) (dtos.StructuredResponse, error) {
	email := registerUserDto.Email
	pass := registerUserDto.Password

	if email == "" || pass == "" {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "Email and Password are required",
			Payload: nil,
		}, errors.New("email or password missing")
	}

	if len(pass) < 6 {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "Password must be at least 6 characters long",
			Payload: nil,
		}, nil
	}

	_, err := s.repo.FindByEmail(ctx, email)
	if err == nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusConflict,
			Message: "email is registered, try a diff email",
			Payload: nil,
		}, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerUserDto.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "Failed to register user",
			Payload: nil,
		}, err
	}

	user := models.User{
		Email:        registerUserDto.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	user, err = s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "Failed to create user",
			Payload: nil,
		}, err
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "Failed to create token",
			Payload: nil,
		}, err
	}

	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusCreated,
		Message: "User registered successfully",
		Payload: map[string]interface{}{
			"id":    user.ID.Hex(),
			"email": user.Email,
			"role":  user.Role,
			"token": token,
		},
	}, nil
}

func (s *AuthService) LoginUser(ctx context.Context, loginUserDto dtos.LoginUserDto) (dtos.StructuredResponse, error) {
	email := loginUserDto.Email
	pass := loginUserDto.Password

	if email == "" || pass == "" {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "Email and Password are required",
			Payload: nil,
		}, errors.New("email or password missing")
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusUnauthorized,
			Message: "Invalid Credentials",
			Payload: nil,
		}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(pass)); err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusConflict,
			Message: "Invalid Credentials",
			Payload: nil,
		}, err
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "Failed to create token",
			Payload: nil,
		}, err
	}

	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "Login successful",
		Payload: map[string]interface{}{
			"id":    user.ID.Hex(),
			"email": user.Email,
			"role":  user.Role,
			"token": token,
		},
	}, nil
}

func (s *AuthService) LogoutUser(ctx context.Context) (dtos.StructuredResponse, error) {
	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "Logout successful",
		Payload: nil,
	}, nil
}
