package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.New()

	if err := router.Run(":8080"); err != nil {
		os.Exit(1)
	}
}
