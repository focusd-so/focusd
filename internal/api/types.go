package api

type Tier string

const (
	TierFree  Tier = "free"
	TierTrial Tier = "trial"
	TierBasic Tier = "basic"
	TierPro   Tier = "pro"
)

type HandshakeNonce struct {
	Nonce     string `gorm:"not null;unique" json:"nonce"`
	CreatedAt int64  `gorm:"not null" json:"created_at"`
	ExpiresAt int64  `gorm:"not null" json:"expires_at"`
}

func (h *HandshakeNonce) TableName() string {
	return "handshake_nonce"
}

type User struct {
	ID            int64        `gorm:"primaryKey;autoIncrement" json:"id"`
	Role          string       `gorm:"default:anonymous;not null" json:"role"`
	CreatedAt     int64        `gorm:"not null" json:"created_at"`
	Devices       []UserDevice `gorm:"foreignKey:UserID" json:"devices"`
	Tier          string       `gorm:"default:trial;not null" json:"tier"`
	TierChangedAt int64        `gorm:"not null" json:"tier_changed_at"`

	// polar
	PolarCustomerID     string `gorm:"not null;unique" json:"polar_customer_id"`
	PolarPastDue        bool   `gorm:"default:false;not null" json:"polar_past_due"`
	PolarSubscriptionID string `gorm:"not null;unique" json:"polar_subscription_id"`
}

func (u *User) TableName() string {
	return "user"
}

type UserDevice struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64  `gorm:"not null" json:"user_id"`
	Fingerprint string `gorm:"not null;unique" json:"fingerprint"`
	CreatedAt   int64  `gorm:"not null" json:"created_at"`
	User        User   `gorm:"foreignKey:UserID" json:"user"`
}

func (u *UserDevice) TableName() string {
	return "user_device"
}
