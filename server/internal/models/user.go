package models

import (
	"gorm.io/gorm"
)

// User 用户表
type User struct {
	gorm.Model        // 自动包含 ID, CreatedAt, UpdatedAt 等字段
	Username   string `gorm:"uniqueIndex;not null" json:"username"`
	Password   string `gorm:"not null" json:"-"` // 密码不通过 JSON 返回给前端
}

// Holding 持仓表 (关联用户)
type Holding struct {
	gorm.Model
	UserID    uint    `gorm:"index;not null" json:"user_id"`
	FundCode  string  `gorm:"not null" json:"fund_code"`
	Amount    float64 `gorm:"not null" json:"amount"` // 持仓金额
	FundName  string  `json:"fund_name"`
	LastPrice string  `json:"last_price"` // 存字符串方便，因为有的是百分比
	Change    string  `json:"change"`     // 涨跌幅
}

// Watchlist 自选表 (关联用户)
type Watchlist struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null" json:"user_id"`
	FundCode string `gorm:"not null" json:"fund_code"`
}
