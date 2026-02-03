package db

import (
	"fmt"
	"fund-tracker-server/internal/models"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	// ✅ 这里已经填好了你提供的密码和数据库名
	// user=postgres (默认用户名)
	// password=05090510Aa (你的密码)
	// dbname=Money_pg (你的数据库名)
	dsn := "host=localhost user=postgres password=05090510Aa dbname=Money_pg port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 数据库连接失败: ", err)
	}

	fmt.Println("✅ 数据库连接成功！")

	// 自动迁移：自动在数据库里创建 User, Holding, Watchlist 表
	err = DB.AutoMigrate(&models.User{}, &models.Holding{}, &models.Watchlist{})
	if err != nil {
		log.Fatal("❌ 数据库迁移失败: ", err)
	}
	fmt.Println("✅ 数据库表结构同步完成")
}
