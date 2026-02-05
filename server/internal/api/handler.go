package api

import (
	"fund-tracker-server/internal/db"
	"fund-tracker-server/internal/models"
	"fund-tracker-server/internal/service"
	"sync" // 🔥 引入 sync 包用于并发控制
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

	// 这里可以复用 FetchFundData 逻辑来计算实时收益，
	// 但为了代码简洁，主要逻辑在 RefreshMarketDB 或由前端触发刷新。
	// 如果需要 GetMyData 也实时，请参考之前的重构逻辑。

	c.JSON(200, gin.H{"holdings": holdings, "watchlist": watchlist})
}

// 添加/更新
func AddFundDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var input struct {
		Code      string  `json:"code"`
		Type      string  `json:"type"`
		Shares    float64 `json:"shares"`
		CostPrice float64 `json:"cost_price"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	fundInfo, err := service.FetchFundData(input.Code)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的基金代码"})
		return
	}

	if input.Type == "holding" {
		var holding models.Holding
		if err := db.DB.Where("user_id = ? AND fund_code = ?", userID, input.Code).First(&holding).Error; err == nil {
			// 加权平均逻辑
			totalShares := holding.Shares + input.Shares
			if totalShares > 0 {
				totalCost := (holding.Shares * holding.CostPrice) + (input.Shares * input.CostPrice)
				holding.CostPrice = totalCost / totalShares
				holding.Shares = totalShares
			}
			holding.FundName = fundInfo.Name
			db.DB.Save(&holding)
		} else {
			db.DB.Create(&models.Holding{
				UserID:    userID,
				FundCode:  input.Code,
				FundName:  fundInfo.Name,
				Shares:    input.Shares,
				CostPrice: input.CostPrice,
			})
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

// 🔥 优化：刷新行情 (并发控制 + 统一返回)
func RefreshMarketDB(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var holdingCodes []string
	var watchCodes []string

	// 获取用户关注的所有代码
	db.DB.Model(&models.Holding{}).Where("user_id = ?", userID).Pluck("fund_code", &holdingCodes)
	db.DB.Model(&models.Watchlist{}).Where("user_id = ?", userID).Pluck("fund_code", &watchCodes)

	// 去重
	uniqueMap := make(map[string]bool)
	for _, code := range holdingCodes {
		uniqueMap[code] = true
	}
	for _, code := range watchCodes {
		uniqueMap[code] = true
	}

	// 准备并发控制
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)  // 🔥 限制最大并发数为 5
	var results []*models.FundInfo // 存储刷新后的数据
	var mu sync.Mutex              // 保护 results 切片的并发写入

	for code := range uniqueMap {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量 (如果满5个则阻塞)

		go func(targetCode string) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 获取最新数据
			data, err := service.FetchFundData(targetCode)
			if err == nil && data != nil {
				// 1. 收集结果
				mu.Lock()
				results = append(results, data)
				mu.Unlock()

				// 2. 更新数据库缓存 (LastPrice 等)
				// 注意：这里仅更新 Holding 表的缓存字段，不影响 shares/cost
				db.DB.Model(&models.Holding{}).
					Where("user_id = ? AND fund_code = ?", userID, targetCode).
					Updates(map[string]interface{}{
						"fund_name":  data.Name,
						"last_price": data.GSZ,
						"change":     data.GSZZL,
					})
			}
		}(code)
	}

	// 等待所有任务完成
	wg.Wait()

	// 🔥 直接返回最新数据列表，前端无需再次调用 GetMyData
	c.JSON(200, gin.H{"data": results})
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
