package main

import (
	"fmt"
	"fund-tracker-server/internal/api"
	"fund-tracker-server/internal/db"
	"fund-tracker-server/internal/service"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func main() {
	db.InitDB()
	r := gin.Default()
	r.Use(Cors())

	r.POST("/register", api.Register)
	r.POST("/login", api.Login)
	r.GET("/detail", func(c *gin.Context) {
		code := c.Query("code")
		detail, err := service.FetchFundDetail(code)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, detail)
	})

	auth := r.Group("/")
	auth.Use(api.AuthMiddleware())
	{
		auth.GET("/my_data", api.GetMyData)
		auth.POST("/add", api.AddFundDB)
		auth.POST("/delete", api.DeleteFundDB)
		auth.GET("/refresh_market", api.RefreshMarketDB)
		auth.GET("/search", api.SearchFundDB)
		// 🔥 已删除: auth.POST("/settle", api.SettleHoldingsDB)
		// 因为现在逻辑是实时计算收益，不需要手动结算接口了
	}

	fmt.Println("🚀 服务端已启动: http://localhost:8080")
	r.Run(":8080")
}
