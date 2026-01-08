package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/models"
)

type WalletService struct {
	container *Container
}

func NewWalletService(c *Container) *WalletService {
	return &WalletService{container: c}
}

type CreateWalletRequest struct {
	Name       string           `json:"name"`
	Type       models.WalletType `json:"type" binding:"required"`
	Tags       []uuid.UUID      `json:"tags,omitempty"`
	GroupID    *uuid.UUID       `json:"group_id,omitempty"`
}

type ImportWalletRequest struct {
	Name       string           `json:"name"`
	Type       models.WalletType `json:"type" binding:"required"`
	PrivateKey string           `json:"private_key" binding:"required"`
	Tags       []uuid.UUID      `json:"tags,omitempty"`
}

type PrepareTransactionRequest struct {
	ChainID     int64  `json:"chain_id" binding:"required"`
	To          string `json:"to" binding:"required"`
	Value       string `json:"value"`
	Data        string `json:"data,omitempty"`
	GasLimit    uint64 `json:"gas_limit,omitempty"`
	GasPrice    string `json:"gas_price,omitempty"`
	MaxFee      string `json:"max_fee,omitempty"`
	MaxPriority string `json:"max_priority,omitempty"`
}

type PreparedTransaction struct {
	UnsignedTx    string `json:"unsigned_tx"`
	TxHash        string `json:"tx_hash"`
	EstimatedGas  uint64 `json:"estimated_gas"`
	GasPrice      string `json:"gas_price"`
	Nonce         uint64 `json:"nonce"`
	SignURL       string `json:"sign_url"` // URL to open in browser for signing
}

func (s *WalletService) List(userID uuid.UUID, walletType string, groupID *uuid.UUID) ([]models.Wallet, error) {
	var wallets []models.Wallet
	query := s.container.DB.Where("user_id = ?", userID).Preload("Tags").Preload("Groups")
	
	if walletType != "" {
		query = query.Where("type = ?", walletType)
	}
	
	if groupID != nil {
		query = query.Joins("JOIN wallet_groups_wallets ON wallet_groups_wallets.wallet_id = wallets.id").
			Where("wallet_groups_wallets.wallet_group_id = ?", groupID)
	}
	
	if err := query.Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (s *WalletService) Create(userID uuid.UUID, req *CreateWalletRequest) (*models.Wallet, error) {
	var wallet *models.Wallet
	var err error

	switch req.Type {
	case models.WalletTypeEVM:
		wallet, err = s.createEVMWallet(userID, req.Name)
	case models.WalletTypeSolana:
		wallet, err = s.createSolanaWallet(userID, req.Name)
	default:
		return nil, errors.New("unsupported wallet type")
	}

	if err != nil {
		return nil, err
	}

	// Add to group if specified
	if req.GroupID != nil {
		var group models.WalletGroup
		if err := s.container.DB.First(&group, req.GroupID).Error; err == nil {
			s.container.DB.Model(&group).Association("Wallets").Append(wallet)
		}
	}

	// Broadcast wallet created event
	s.container.WSHub.BroadcastToUser(userID.String(), "wallet:created", wallet)

	return wallet, nil
}

func (s *WalletService) createEVMWallet(userID uuid.UUID, name string) (*models.Wallet, error) {
	// Generate new private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	// Get address
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	
	// Encrypt private key
	privateKeyBytes := crypto.FromECDSA(privateKey)
	encryptedKey, err := s.encryptPrivateKey(hex.EncodeToString(privateKeyBytes))
	if err != nil {
		return nil, err
	}

	wallet := &models.Wallet{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         name,
		Address:      address,
		Type:         models.WalletTypeEVM,
		EncryptedKey: encryptedKey,
		PublicKey:    hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)),
		IsImported:   false,
		Balance:      "0",
	}

	if err := s.container.DB.Create(wallet).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *WalletService) createSolanaWallet(userID uuid.UUID, name string) (*models.Wallet, error) {
	// For Solana, we'll use ed25519 keypair
	// This is a simplified version - in production use proper Solana SDK
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	privateKeyBytes := crypto.FromECDSA(privateKey)
	encryptedKey, err := s.encryptPrivateKey(hex.EncodeToString(privateKeyBytes))
	if err != nil {
		return nil, err
	}

	wallet := &models.Wallet{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         name,
		Address:      address,
		Type:         models.WalletTypeSolana,
		EncryptedKey: encryptedKey,
		IsImported:   false,
		Balance:      "0",
	}

	if err := s.container.DB.Create(wallet).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *WalletService) Import(userID uuid.UUID, req *ImportWalletRequest) (*models.Wallet, error) {
	// Validate and get address from private key
	privateKeyBytes, err := hex.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, errors.New("invalid private key format")
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, errors.New("invalid private key")
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	// Check if wallet already exists
	var existing models.Wallet
	if err := s.container.DB.Where("address = ?", address).First(&existing).Error; err == nil {
		return nil, errors.New("wallet already imported")
	}

	// Encrypt private key
	encryptedKey, err := s.encryptPrivateKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}

	wallet := &models.Wallet{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		Address:      address,
		Type:         req.Type,
		EncryptedKey: encryptedKey,
		PublicKey:    hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)),
		IsImported:   true,
		Balance:      "0",
	}

	if err := s.container.DB.Create(wallet).Error; err != nil {
		return nil, err
	}

	// Sync balance
	go s.SyncBalance(wallet.ID)

	return wallet, nil
}

func (s *WalletService) Get(userID, walletID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := s.container.DB.Where("id = ? AND user_id = ?", walletID, userID).
		Preload("Tags").
		Preload("Groups").
		Preload("LinkedAccounts").
		First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (s *WalletService) Update(userID, walletID uuid.UUID, updates map[string]interface{}) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := s.container.DB.Where("id = ? AND user_id = ?", walletID, userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	// Only allow certain fields to be updated
	allowedFields := map[string]bool{"name": true}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}

	if err := s.container.DB.Model(&wallet).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (s *WalletService) Delete(userID, walletID uuid.UUID) error {
	return s.container.DB.Where("id = ? AND user_id = ?", walletID, userID).Delete(&models.Wallet{}).Error
}

func (s *WalletService) GetBalance(walletID uuid.UUID) (*models.WalletBalance, error) {
	var wallet models.Wallet
	if err := s.container.DB.First(&wallet, walletID).Error; err != nil {
		return nil, err
	}

	// Try to get from cache first
	ctx := context.Background()
	cacheKey := fmt.Sprintf("wallet:balance:%s", wallet.Address)
	cached, err := s.container.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var balance models.WalletBalance
		if json.Unmarshal([]byte(cached), &balance) == nil {
			return &balance, nil
		}
	}

	// Fetch fresh balance
	balance, err := s.fetchBalance(&wallet)
	if err != nil {
		return nil, err
	}

	// Cache for 30 seconds
	if data, err := json.Marshal(balance); err == nil {
		s.container.Redis.Set(ctx, cacheKey, data, 30*time.Second)
	}

	return balance, nil
}

func (s *WalletService) fetchBalance(wallet *models.Wallet) (*models.WalletBalance, error) {
	balance := &models.WalletBalance{
		Address:   wallet.Address,
		UpdatedAt: time.Now(),
	}

	if wallet.Type == models.WalletTypeEVM {
		// For demo, using public Ethereum RPC
		client, err := ethclient.Dial("https://eth.llamarpc.com")
		if err != nil {
			return balance, nil // Return empty balance on error
		}
		defer client.Close()

		address := common.HexToAddress(wallet.Address)
		balanceWei, err := client.BalanceAt(context.Background(), address, nil)
		if err != nil {
			return balance, nil
		}

		balance.NativeBalance = balanceWei.String()
	}

	return balance, nil
}

func (s *WalletService) SyncBalance(walletID uuid.UUID) error {
	balance, err := s.GetBalance(walletID)
	if err != nil {
		return err
	}

	return s.container.DB.Model(&models.Wallet{}).Where("id = ?", walletID).Updates(map[string]interface{}{
		"balance":           balance.NativeBalance,
		"last_balance_sync": time.Now(),
	}).Error
}

func (s *WalletService) GetTransactions(userID, walletID uuid.UUID, limit, offset int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	query := s.container.DB.Model(&models.Transaction{}).Where("wallet_id = ?", walletID)
	query.Count(&total)

	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (s *WalletService) PrepareTransaction(userID, walletID uuid.UUID, req *PrepareTransactionRequest) (*PreparedTransaction, error) {
	var wallet models.Wallet
	if err := s.container.DB.Where("id = ? AND user_id = ?", walletID, userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	// Get RPC URL for chain
	rpcURL := s.getRPCURL(req.ChainID)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get nonce
	fromAddress := common.HexToAddress(wallet.Address)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	// Get gas price if not provided
	var gasPrice *big.Int
	if req.GasPrice != "" {
		gasPrice = new(big.Int)
		gasPrice.SetString(req.GasPrice, 10)
	} else {
		gasPrice, err = client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Parse value
	value := new(big.Int)
	if req.Value != "" {
		value.SetString(req.Value, 10)
	}

	// Parse data
	var data []byte
	if req.Data != "" {
		data, _ = hex.DecodeString(req.Data)
	}

	// Estimate gas if not provided
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000 // Default for simple transfers
	}

	// Create unsigned transaction
	toAddress := common.HexToAddress(req.To)
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	// Serialize transaction
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	prepared := &PreparedTransaction{
		UnsignedTx:   hex.EncodeToString(txBytes),
		TxHash:       tx.Hash().Hex(),
		EstimatedGas: gasLimit,
		GasPrice:     gasPrice.String(),
		Nonce:        nonce,
		SignURL:      fmt.Sprintf("/browser/sign?wallet=%s&tx=%s", wallet.Address, hex.EncodeToString(txBytes)),
	}

	return prepared, nil
}

func (s *WalletService) BulkCreate(userID uuid.UUID, count int, walletType models.WalletType, groupID *uuid.UUID) ([]models.Wallet, error) {
	var wallets []models.Wallet

	for i := 0; i < count; i++ {
		req := &CreateWalletRequest{
			Name:    fmt.Sprintf("Wallet %d", i+1),
			Type:    walletType,
			GroupID: groupID,
		}
		wallet, err := s.Create(userID, req)
		if err != nil {
			continue
		}
		wallets = append(wallets, *wallet)
	}

	return wallets, nil
}

func (s *WalletService) encryptPrivateKey(privateKey string) (string, error) {
	key := []byte(s.container.Config.EncryptionKey)
	if len(key) < 32 {
		key = append(key, make([]byte, 32-len(key))...)
	}
	key = key[:32]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(privateKey), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (s *WalletService) decryptPrivateKey(encrypted string) (string, error) {
	key := []byte(s.container.Config.EncryptionKey)
	if len(key) < 32 {
		key = append(key, make([]byte, 32-len(key))...)
	}
	key = key[:32]

	ciphertext, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (s *WalletService) getPrivateKey(wallet *models.Wallet) (*ecdsa.PrivateKey, error) {
	decrypted, err := s.decryptPrivateKey(wallet.EncryptedKey)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := hex.DecodeString(decrypted)
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(privateKeyBytes)
}

func (s *WalletService) getRPCURL(chainID int64) string {
	rpcURLs := map[int64]string{
		1:     "https://eth.llamarpc.com",
		137:   "https://polygon-rpc.com",
		42161: "https://arb1.arbitrum.io/rpc",
		10:    "https://mainnet.optimism.io",
		8453:  "https://mainnet.base.org",
	}
	
	if url, ok := rpcURLs[chainID]; ok {
		return url
	}
	return "https://eth.llamarpc.com"
}
