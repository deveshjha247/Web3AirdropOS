package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletType string

const (
	WalletTypeEVM    WalletType = "evm"
	WalletTypeSolana WalletType = "solana"
)

type Wallet struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Name            string         `gorm:"size:100" json:"name"`
	Address         string         `gorm:"size:100;not null;uniqueIndex" json:"address"`
	Type            WalletType     `gorm:"size:20;not null" json:"type"`
	EncryptedKey    string         `gorm:"type:text" json:"-"` // Encrypted private key (stored securely)
	PublicKey       string         `gorm:"size:200" json:"public_key"`
	IsImported      bool           `gorm:"default:false" json:"is_imported"`
	IsWatchOnly     bool           `gorm:"default:false" json:"is_watch_only"`
	Balance         string         `gorm:"size:100;default:'0'" json:"balance"`
	LastBalanceSync time.Time      `json:"last_balance_sync"`
	Tags            []WalletTag    `gorm:"many2many:wallet_wallet_tags;" json:"tags"`
	Groups          []WalletGroup  `gorm:"many2many:wallet_groups_wallets;" json:"groups"`
	LinkedAccounts  []PlatformAccount `gorm:"foreignKey:WalletID" json:"linked_accounts"`
	Transactions    []Transaction  `gorm:"foreignKey:WalletID" json:"transactions,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type WalletTag struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Color     string    `gorm:"size:20" json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type WalletGroup struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Wallets     []Wallet   `gorm:"many2many:wallet_groups_wallets;" json:"wallets"`
	CampaignID  *uuid.UUID `gorm:"type:uuid" json:"campaign_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Transaction struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WalletID        uuid.UUID `gorm:"type:uuid;not null" json:"wallet_id"`
	Hash            string    `gorm:"size:100;uniqueIndex" json:"hash"`
	ChainID         int       `json:"chain_id"`
	FromAddress     string    `gorm:"size:100" json:"from_address"`
	ToAddress       string    `gorm:"size:100" json:"to_address"`
	Value           string    `gorm:"size:100" json:"value"`
	GasUsed         string    `gorm:"size:50" json:"gas_used"`
	GasPrice        string    `gorm:"size:50" json:"gas_price"`
	Status          string    `gorm:"size:20" json:"status"` // pending, success, failed
	BlockNumber     int64     `json:"block_number"`
	Timestamp       time.Time `json:"timestamp"`
	RawTransaction  string    `gorm:"type:text" json:"raw_transaction,omitempty"`
	DecodedData     string    `gorm:"type:jsonb" json:"decoded_data,omitempty"`
	TaskExecutionID *uuid.UUID `gorm:"type:uuid" json:"task_execution_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// WalletBalance represents cached balance info
type WalletBalance struct {
	Address     string            `json:"address"`
	NativeBalance string          `json:"native_balance"`
	Tokens      []TokenBalance    `json:"tokens"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type TokenBalance struct {
	ContractAddress string `json:"contract_address"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	Balance         string `json:"balance"`
	Decimals        int    `json:"decimals"`
}
