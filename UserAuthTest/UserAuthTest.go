package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Print("Code is ", " starting.\n")
	router := gin.Default()
	err := router.Run("localhost:9090")
	if err != nil {
		return
	}
}
