package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/nanami9426/imgo/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func get_dsn() string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env文件读取失败")
	}
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_HOST"),
		os.Getenv("MYSQL_PORT"),
		os.Getenv("MYSQL_DB"),
	)
	return dsn
}

func main() {
	db, err := gorm.Open(mysql.Open(get_dsn()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.UserBasic{})

	// Create
	// user := &models.UserBasic{}
	// user.Name = "测试"
	// db.Create(user)
	// psw, _ := rand.Int(rand.Reader, big.NewInt(100_000_000))
	// psw_str := fmt.Sprintf("%d", psw)
	// db.Model(user).Update("Password", psw_str)
}
