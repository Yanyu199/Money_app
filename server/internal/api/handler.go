package api

import (
	"fmt" // 🔥 记得确认导入了 fmt
	"fund-tracker-server/internal/db"
	"fund-tracker-server/internal/models"
	"fund-tracker-server/internal/service"
	"strconv" // 🔥 记得确认导入了 strconv
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("my_super_secret_key_2026")

// 注册
func Register(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := models.User{Username: input.Username, Password: string(hashedPwd)}
	if result := db.DB.Create(&user); result.Error != nil {
		c.JSON(500, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(200, gin.H{"message": "注册成功"})
}

// 登录
func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := db.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"error": "用户不存在"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	c.JSON(200, gin.H{"token": tokenString, "username": user.Username})
}

// 获取数据
func GetMyData(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var holdings []models.Holding
	var watchlist []models.Watchlist
	db.DB.Where("user_id = ?", userID).Find(&holdings)
	db.DB.Where("user_id = ?", userID).Find(&watchlist)
	c.JSON(200, gin.H{"holdings": holdings, "watchlist": watchlist})
}

// 添加/更新
func AddFundDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var input struct {
		Code   string  `json:"code"`
		Type   string  `json:"type"`
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if _, err := service.FetchFundData(input.Code); err != nil {
		c.JSON(400, gin.H{"error": "无效的基金代码"})
		return
	}
	if input.Type == "holding" {
		var holding models.Holding
		if err := db.DB.Where("user_id = ? AND fund_code = ?", userID, input.Code).First(&holding).Error; err == nil {
			holding.Amount = input.Amount
			db.DB.Save(&holding)
		} else {
			db.DB.Create(&models.Holding{UserID: userID, FundCode: input.Code, Amount: input.Amount})
		}
	} else {
		var count int64
		db.DB.Model(&models.Watchlist{}).Where("user_id = ? AND fund_code = ?", userID, input.Code).Count(&count)
		if count == 0 {
			db.DB.Create(&models.Watchlist{UserID: userID, FundCode: input.Code})
		}
	}
	c.JSON(200, gin.H{"success": true})
}

// 删除
func DeleteFundDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var input struct {
		Code string `json:"code"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if input.Type == "holding" {
		db.DB.Unscoped().Where("user_id = ? AND fund_code = ?", userID, input.Code).Delete(&models.Holding{})
	} else {
		db.DB.Unscoped().Where("user_id = ? AND fund_code = ?", userID, input.Code).Delete(&models.Watchlist{})
	}
	c.JSON(200, gin.H{"success": true})
}

// 搜索接口
func SearchFundDB(c *gin.Context) {
	keyword := c.Query("key")
	if keyword == "" {
		c.JSON(400, gin.H{"error": "missing key"})
		return
	}
	results, err := service.SearchFund(keyword)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": results})
}

// 刷新行情
func RefreshMarketDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var holdingCodes []string
	var watchCodes []string
	db.DB.Model(&models.Holding{}).Where("user_id = ?", userID).Pluck("fund_code", &holdingCodes)
	db.DB.Model(&models.Watchlist{}).Where("user_id = ?", userID).Pluck("fund_code", &watchCodes)
	uniqueMap := make(map[string]bool)
	for _, code := range holdingCodes {
		uniqueMap[code] = true
	}
	for _, code := range watchCodes {
		uniqueMap[code] = true
	}

	var results []interface{}
	for code := range uniqueMap {
		data, err := service.FetchFundData(code)
		if err == nil {
			results = append(results, data)
			go func(uid uint, c string, d *models.FundInfo) {
				db.DB.Model(&models.Holding{}).
					Where("user_id = ? AND fund_code = ?", uid, c).
					Updates(map[string]interface{}{
						"fund_name":  d.Name,
						"last_price": d.GSZ,
						"change":     d.GSZZL,
					})
			}(userID, code, data)
		}
	}
	c.JSON(200, gin.H{"data": results})
}

// 🔥 新增：一键结算（把收益更新进持仓本金）
func SettleHoldingsDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	// 1. 获取所有持仓
	var holdings []models.Holding
	if err := db.DB.Where("user_id = ?", userID).Find(&holdings).Error; err != nil {
		c.JSON(500, gin.H{"error": "获取持仓失败"})
		return
	}

	updatedCount := 0
	totalDiff := 0.0

	// 2. 遍历每一个基金
	for _, h := range holdings {
		// 这里的 FetchFundData 已经包含了“优先取场内/官方确权”的逻辑
		fundInfo, err := service.FetchFundData(h.FundCode)

		if err == nil && fundInfo != nil {
			// 解析涨跌幅
			rateStr := fundInfo.GSZZL
			rate, _ := strconv.ParseFloat(rateStr, 64)

			// 如果涨跌幅不是 0，就开始结算
			if rate != 0 {
				oldAmount := h.Amount
				// 计算收益： 本金 * (涨跌幅 / 100)
				profit := oldAmount * (rate / 100.0)
				newAmount := oldAmount + profit

				// 更新数据库
				h.Amount = newAmount
				h.FundName = fundInfo.Name
				h.LastPrice = fundInfo.GSZ
				h.Change = fundInfo.GSZZL

				db.DB.Save(&h)

				updatedCount++
				totalDiff += profit
			}
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": fmt.Sprintf("已结算 %d 支基金\n总资产变动: %+.2f", updatedCount, totalDiff),
	})
}

// 中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "未登录"})
			return
		}
		token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user_id", uint(claims["user_id"].(float64)))
			c.Next()
		} else {
			c.AbortWithStatusJSON(401, gin.H{"error": "Token 无效"})
		}
	}
}
