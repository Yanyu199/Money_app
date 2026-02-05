package models

import (
	"gorm.io/gorm"
)

// User 用户表
type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Password string `gorm:"not null" json:"-"`
}

// Holding 持仓表
type Holding struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null" json:"user_id"`
	FundCode string `gorm:"not null" json:"fund_code"`
	FundName string `json:"fund_name"`

	// 🔥 修改：增加 ;default:0 以兼容旧数据
	Shares    float64 `gorm:"not null;default:0" json:"shares"`     // 持有份额
	CostPrice float64 `gorm:"not null;default:0" json:"cost_price"` // 平均成本单价

	// 缓存字段
	LastPrice string `json:"last_price"`
	Change    string `json:"change"`

	// 动态计算字段
	TotalValue  float64 `gorm:"-" json:"total_value"`
	TotalReturn float64 `gorm:"-" json:"total_return"`
	DayReturn   float64 `gorm:"-" json:"day_return"`
}

// Watchlist 自选表
type Watchlist struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null" json:"user_id"`
	FundCode string `gorm:"not null" json:"fund_code"`
}
