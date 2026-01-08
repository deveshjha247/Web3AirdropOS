package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

// SecretType represents the type of secret
type SecretType string

const (
	SecretTypeAPIKey      SecretType = "api_key"
	SecretTypeToken       SecretType = "token"
	SecretTypePassword    SecretType = "password"
	SecretTypePrivateKey  SecretType = "private_key"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeCredentials SecretType = "credentials"
)

// Common errors
var (
	ErrSecretNotFound   = errors.New("secret not found")
	ErrInvalidKey       = errors.New("invalid encryption key")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrSecretExists     = errors.New("secret already exists")
)

// Secret represents a stored secret
type Secret struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	KeyType        SecretType     `gorm:"size:50;not null" json:"key_type"`
	EncryptedValue string         `gorm:"type:text;not null" json:"-"` // Never expose
	IV             string         `gorm:"size:32;not null" json:"-"`   // Initialization vector
	Metadata       string         `gorm:"type:jsonb" json:"metadata,omitempty"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
	LastAccessedAt *time.Time     `json:"last_accessed_at,omitempty"`
	AccessCount    int            `gorm:"default:0" json:"access_count"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Vault manages encrypted secrets
type Vault struct {
	db         *gorm.DB
	masterKey  []byte // 32-byte key for AES-256
	keyDeriver func(password, salt []byte) []byte
}

// Config for the vault
type Config struct {
	MasterKey string // 32-byte hex-encoded master key
}

// NewVault creates a new secrets vault
func NewVault(db *gorm.DB, config Config) (*Vault, error) {
	// Decode or derive master key
	masterKey, err := hex.DecodeString(config.MasterKey)
	if err != nil || len(masterKey) != 32 {
		// If not valid hex, derive from password using Argon2
		salt := []byte("web3airdropos-vault-salt") // Fixed salt for key derivation
		masterKey = argon2.IDKey([]byte(config.MasterKey), salt, 3, 64*1024, 4, 32)
	}

	return &Vault{
		db:        db,
		masterKey: masterKey,
		keyDeriver: func(password, salt []byte) []byte {
			return argon2.IDKey(password, salt, 3, 64*1024, 4, 32)
		},
	}, nil
}

// Store stores an encrypted secret
func (v *Vault) Store(ctx context.Context, userID uuid.UUID, name string, value string, keyType SecretType, metadata map[string]interface{}) (*Secret, error) {
	// Check if secret already exists
	var existing Secret
	if err := v.db.Where("user_id = ? AND name = ?", userID, name).First(&existing).Error; err == nil {
		return nil, ErrSecretExists
	}

	// Generate unique IV for this secret
	iv := make([]byte, 12) // GCM standard nonce size
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Derive a unique key for this secret using user ID as additional salt
	secretKey := v.deriveSecretKey(userID, name)

	// Encrypt the value
	encrypted, err := v.encrypt([]byte(value), secretKey, iv)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	secret := &Secret{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           name,
		KeyType:        keyType,
		EncryptedValue: base64.StdEncoding.EncodeToString(encrypted),
		IV:             hex.EncodeToString(iv),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if metadata != nil {
		// Don't store sensitive data in metadata
		metadataJSON, _ := json.Marshal(metadata)
		secret.Metadata = string(metadataJSON)
	}

	if err := v.db.Create(secret).Error; err != nil {
		return nil, err
	}

	return secret, nil
}

// Retrieve retrieves and decrypts a secret
func (v *Vault) Retrieve(ctx context.Context, userID uuid.UUID, name string) (string, error) {
	var secret Secret
	if err := v.db.Where("user_id = ? AND name = ?", userID, name).First(&secret).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	// Check expiration
	if secret.ExpiresAt != nil && time.Now().After(*secret.ExpiresAt) {
		return "", errors.New("secret has expired")
	}

	// Decode IV
	iv, err := hex.DecodeString(secret.IV)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	// Decode encrypted value
	encrypted, err := base64.StdEncoding.DecodeString(secret.EncryptedValue)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	// Derive the secret key
	secretKey := v.deriveSecretKey(userID, name)

	// Decrypt
	decrypted, err := v.decrypt(encrypted, secretKey, iv)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	// Update access tracking
	now := time.Now()
	v.db.Model(&secret).Updates(map[string]interface{}{
		"last_accessed_at": now,
		"access_count":     gorm.Expr("access_count + 1"),
	})

	return string(decrypted), nil
}

// RetrieveByID retrieves a secret by ID
func (v *Vault) RetrieveByID(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (string, *Secret, error) {
	var secret Secret
	if err := v.db.Where("id = ? AND user_id = ?", secretID, userID).First(&secret).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrSecretNotFound
		}
		return "", nil, err
	}

	value, err := v.Retrieve(ctx, userID, secret.Name)
	if err != nil {
		return "", nil, err
	}

	return value, &secret, nil
}

// Update updates a secret's value
func (v *Vault) Update(ctx context.Context, userID uuid.UUID, name string, newValue string) error {
	var secret Secret
	if err := v.db.Where("user_id = ? AND name = ?", userID, name).First(&secret).Error; err != nil {
		return ErrSecretNotFound
	}

	// Generate new IV
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return fmt.Errorf("failed to generate IV: %w", err)
	}

	// Derive secret key
	secretKey := v.deriveSecretKey(userID, name)

	// Encrypt new value
	encrypted, err := v.encrypt([]byte(newValue), secretKey, iv)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	secret.EncryptedValue = base64.StdEncoding.EncodeToString(encrypted)
	secret.IV = hex.EncodeToString(iv)
	secret.UpdatedAt = time.Now()

	return v.db.Save(&secret).Error
}

// Delete soft-deletes a secret
func (v *Vault) Delete(ctx context.Context, userID uuid.UUID, name string) error {
	result := v.db.Where("user_id = ? AND name = ?", userID, name).Delete(&Secret{})
	if result.RowsAffected == 0 {
		return ErrSecretNotFound
	}
	return result.Error
}

// List lists all secrets for a user (without decrypted values)
func (v *Vault) List(ctx context.Context, userID uuid.UUID) ([]Secret, error) {
	var secrets []Secret
	if err := v.db.Where("user_id = ?", userID).Find(&secrets).Error; err != nil {
		return nil, err
	}

	// Clear sensitive fields
	for i := range secrets {
		secrets[i].EncryptedValue = ""
		secrets[i].IV = ""
	}

	return secrets, nil
}

// ListByType lists secrets by type
func (v *Vault) ListByType(ctx context.Context, userID uuid.UUID, keyType SecretType) ([]Secret, error) {
	var secrets []Secret
	if err := v.db.Where("user_id = ? AND key_type = ?", userID, keyType).Find(&secrets).Error; err != nil {
		return nil, err
	}

	// Clear sensitive fields
	for i := range secrets {
		secrets[i].EncryptedValue = ""
		secrets[i].IV = ""
	}

	return secrets, nil
}

// Exists checks if a secret exists
func (v *Vault) Exists(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	var count int64
	err := v.db.Model(&Secret{}).Where("user_id = ? AND name = ?", userID, name).Count(&count).Error
	return count > 0, err
}

// SetExpiration sets an expiration time for a secret
func (v *Vault) SetExpiration(ctx context.Context, userID uuid.UUID, name string, expiresAt time.Time) error {
	result := v.db.Model(&Secret{}).
		Where("user_id = ? AND name = ?", userID, name).
		Update("expires_at", expiresAt)
	if result.RowsAffected == 0 {
		return ErrSecretNotFound
	}
	return result.Error
}

// RotateKey re-encrypts all secrets with a new key (for key rotation)
func (v *Vault) RotateKey(ctx context.Context, userID uuid.UUID, newMasterKey []byte) error {
	// Get all secrets for user
	var secrets []Secret
	if err := v.db.Where("user_id = ?", userID).Find(&secrets).Error; err != nil {
		return err
	}

	// Create new vault instance with new key for encryption
	newVault := &Vault{
		db:        v.db,
		masterKey: newMasterKey,
	}

	// Re-encrypt each secret
	for _, secret := range secrets {
		// Decrypt with old key
		iv, _ := hex.DecodeString(secret.IV)
		encrypted, _ := base64.StdEncoding.DecodeString(secret.EncryptedValue)
		secretKey := v.deriveSecretKey(userID, secret.Name)
		decrypted, err := v.decrypt(encrypted, secretKey, iv)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret %s: %w", secret.Name, err)
		}

		// Generate new IV
		newIV := make([]byte, 12)
		if _, err := io.ReadFull(rand.Reader, newIV); err != nil {
			return err
		}

		// Encrypt with new key
		newSecretKey := newVault.deriveSecretKey(userID, secret.Name)
		newEncrypted, err := newVault.encrypt(decrypted, newSecretKey, newIV)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret %s: %w", secret.Name, err)
		}

		// Update
		secret.EncryptedValue = base64.StdEncoding.EncodeToString(newEncrypted)
		secret.IV = hex.EncodeToString(newIV)
		secret.UpdatedAt = time.Now()

		if err := v.db.Save(&secret).Error; err != nil {
			return err
		}
	}

	// Update vault master key
	v.masterKey = newMasterKey
	return nil
}

// deriveSecretKey derives a unique key for each secret
func (v *Vault) deriveSecretKey(userID uuid.UUID, name string) []byte {
	salt := append(userID[:], []byte(name)...)
	return v.keyDeriver(v.masterKey, salt)
}

// encrypt encrypts data using AES-256-GCM
func (v *Vault) encrypt(plaintext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (v *Vault) decrypt(ciphertext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// CleanupExpired removes expired secrets
func (v *Vault) CleanupExpired(ctx context.Context) (int64, error) {
	result := v.db.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&Secret{})
	return result.RowsAffected, result.Error
}
