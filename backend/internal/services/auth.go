package services

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/web3airdropos/backend/internal/auth"
	"github.com/web3airdropos/backend/internal/models"
)

type AuthService struct {
	container      *Container
	productionAuth *auth.AuthService // Production auth with token family rotation
}

func NewAuthService(c *Container) *AuthService {
	return &AuthService{container: c}
}

// SetProductionAuth sets the production auth service for token family rotation
func (s *AuthService) SetProductionAuth(authService *auth.AuthService) {
	s.productionAuth = authService
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

func (s *AuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	ctx := context.Background()

	// Use production auth if available
	if s.productionAuth != nil {
		result, err := s.productionAuth.Register(ctx, req.Email, req.Password, req.Name, "")
		if err != nil {
			return nil, err
		}

		// Get user from database
		var user models.User
		if err := s.container.DB.First(&user, result.UserID).Error; err != nil {
			return nil, err
		}
		user.PasswordHash = ""

		return &AuthResponse{
			User:         &user,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		}, nil
	}

	// Fallback to basic auth
	return s.registerBasic(req)
}

func (s *AuthService) registerBasic(req *RegisterRequest) (*AuthResponse, error) {
	// Check if user exists
	var existing models.User
	if err := s.container.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
	}

	if err := s.container.DB.Create(user).Error; err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateTokens(user)
}

func (s *AuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	ctx := context.Background()

	// Use production auth if available
	if s.productionAuth != nil {
		result, err := s.productionAuth.Login(ctx, req.Email, req.Password, "", "")
		if err != nil {
			return nil, err
		}

		// Get user from database
		var user models.User
		if err := s.container.DB.First(&user, result.UserID).Error; err != nil {
			return nil, err
		}
		user.PasswordHash = ""

		return &AuthResponse{
			User:         &user,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		}, nil
	}

	// Fallback to basic auth
	return s.loginBasic(req)
}

func (s *AuthService) loginBasic(req *LoginRequest) (*AuthResponse, error) {
	var user models.User
	if err := s.container.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	s.container.DB.Model(&user).Update("last_login_at", time.Now())

	return s.generateTokens(&user)
}

func (s *AuthService) RefreshToken(refreshToken string) (*AuthResponse, error) {
	ctx := context.Background()

	// Use production auth if available
	if s.productionAuth != nil {
		result, err := s.productionAuth.RefreshToken(ctx, refreshToken)
		if err != nil {
			return nil, err
		}

		// Get user from database
		var user models.User
		if err := s.container.DB.First(&user, result.UserID).Error; err != nil {
			return nil, err
		}
		user.PasswordHash = ""

		return &AuthResponse{
			User:         &user,
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		}, nil
	}

	// Fallback to basic refresh
	return s.refreshTokenBasic(refreshToken)
}

func (s *AuthService) refreshTokenBasic(refreshToken string) (*AuthResponse, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.container.Config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	// Get user
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := s.container.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	return s.generateTokens(&user)
}

func (s *AuthService) generateTokens(user *models.User) (*AuthResponse, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	// Access token
	accessClaims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     expiresAt.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.container.Config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Refresh token (7 days)
	refreshClaims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.container.Config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &AuthResponse{
		User:         user,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
	}, nil
}
