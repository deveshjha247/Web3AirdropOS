package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/models"
)

// Common errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrTokenFamilyCompromised = errors.New("token family compromised - all sessions revoked")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims represents JWT claims for access tokens
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	TokenType TokenType `json:"token_type"`
	SessionID uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents JWT claims for refresh tokens
type RefreshTokenClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	TokenType TokenType `json:"token_type"`
	FamilyID  uuid.UUID `json:"family_id"` // Token rotation detection
	jwt.RegisteredClaims
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null"`
	TokenHash  string     `gorm:"size:64;uniqueIndex;not null"`
	FamilyID   uuid.UUID  `gorm:"type:uuid;not null"`
	ExpiresAt  time.Time  `gorm:"not null"`
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time
	IPAddress  string     `gorm:"size:50"`
	UserAgent  string     `gorm:"size:500"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// AuthService handles authentication
type AuthService struct {
	db                   *gorm.DB
	jwtSecret            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:                   db,
		jwtSecret:            []byte(jwtSecret),
		accessTokenDuration:  15 * time.Minute,  // Short-lived access tokens
		refreshTokenDuration: 7 * 24 * time.Hour, // 7-day refresh tokens
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User        *models.User `json:"user"`
	Tokens      TokenPair    `json:"tokens"`
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest, ipAddress, userAgent string) (*AuthResponse, error) {
	// Check if user exists
	var existing models.User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailExists
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
		IsActive:     true,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &AuthResponse{
		User:   user,
		Tokens: *tokens,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*AuthResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	s.db.Model(&user).Update("last_login_at", now)

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, &user, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &AuthResponse{
		User:   &user,
		Tokens: *tokens,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString, ipAddress, userAgent string) (*TokenPair, error) {
	// Parse and validate refresh token
	claims := &RefreshTokenClaims{}
	token, err := jwt.ParseWithClaims(refreshTokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}

	// Hash the token for lookup
	tokenHash := hashToken(refreshTokenString)

	// Find the stored refresh token
	var storedToken RefreshToken
	if err := s.db.Where("token_hash = ?", tokenHash).First(&storedToken).Error; err != nil {
		return nil, ErrInvalidToken
	}

	// Check if token was revoked
	if storedToken.RevokedAt != nil {
		// Token reuse detected! Revoke entire family
		s.revokeTokenFamily(ctx, storedToken.FamilyID)
		return nil, ErrTokenFamilyCompromised
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	// Get user
	var user models.User
	if err := s.db.First(&user, claims.UserID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	// Revoke the old refresh token
	now := time.Now()
	storedToken.RevokedAt = &now

	// Generate new token pair (same family for rotation tracking)
	newTokens, newStoredToken, err := s.generateTokenPairWithFamily(ctx, &user, storedToken.FamilyID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	// Update old token with reference to new one
	storedToken.ReplacedBy = &newStoredToken.ID
	s.db.Save(&storedToken)

	return newTokens, nil
}

// ValidateAccessToken validates an access token and returns claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Logout revokes all tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return s.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// LogoutSession revokes tokens for a specific session
func (s *AuthService) LogoutSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	// Sessions are tracked via token families
	// Find and revoke the family
	var token RefreshToken
	if err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&token).Error; err != nil {
		return err
	}
	return s.revokeTokenFamily(ctx, token.FamilyID)
}

// generateTokenPair creates a new access/refresh token pair
func (s *AuthService) generateTokenPair(ctx context.Context, user *models.User, ipAddress, userAgent string) (*TokenPair, error) {
	familyID := uuid.New() // New token family for new login
	tokens, _, err := s.generateTokenPairWithFamily(ctx, user, familyID, ipAddress, userAgent)
	return tokens, err
}

// generateTokenPairWithFamily creates tokens with a specific family ID
func (s *AuthService) generateTokenPairWithFamily(ctx context.Context, user *models.User, familyID uuid.UUID, ipAddress, userAgent string) (*TokenPair, *RefreshToken, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessTokenDuration)
	refreshExpiry := now.Add(s.refreshTokenDuration)
	sessionID := uuid.New()

	// Create access token
	accessClaims := Claims{
		UserID:    user.ID,
		Email:     user.Email,
		TokenType: TokenTypeAccess,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, nil, err
	}

	// Create refresh token
	refreshClaims := RefreshTokenClaims{
		UserID:    user.ID,
		TokenType: TokenTypeRefresh,
		FamilyID:  familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        sessionID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, nil, err
	}

	// Store refresh token hash
	storedToken := &RefreshToken{
		ID:        sessionID,
		UserID:    user.ID,
		TokenHash: hashToken(refreshTokenString),
		FamilyID:  familyID,
		ExpiresAt: refreshExpiry,
		CreatedAt: now,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.db.Create(storedToken).Error; err != nil {
		return nil, nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
		TokenType:    "Bearer",
	}, storedToken, nil
}

// revokeTokenFamily revokes all tokens in a family
func (s *AuthService) revokeTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	now := time.Now()
	return s.db.Model(&RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

// CleanupExpiredTokens removes expired tokens
func (s *AuthService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&RefreshToken{})
	return result.RowsAffected, result.Error
}

// GetActiveSessions returns active sessions for a user
func (s *AuthService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	var tokens []RefreshToken
	err := s.db.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// generateRandomToken generates a cryptographically secure random token
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
