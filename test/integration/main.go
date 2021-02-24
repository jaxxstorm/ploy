package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	var listenPort string

	port, defined := os.LookupEnv("LISTEN_PORT")

	if defined {
		listenPort = port
	} else {
		listenPort = "80"
	}

	r.Run(fmt.Sprintf(":%s", listenPort))
}

