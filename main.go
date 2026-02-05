package main

import (
	"github.com/nanami9426/imgo/internal/router"
	"github.com/nanami9426/imgo/internal/utils"
)

func main() {
	utils.InitConfig()
	r := router.Router()
	r.Run(":8000")
}
