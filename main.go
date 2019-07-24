package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := localitiesHandler()

	router.Run()
}

func getLocalities(c *gin.Context) {
	c.AbortWithStatus(http.StatusOK)
}

func getLocalitiesByUF(c *gin.Context) {
	c.AbortWithStatus(http.StatusOK)
}
func localitiesHandler() *gin.Engine {
	router := gin.Default()

	v1 := router.Group("/v1")

	v1.GET("/localities", getLocalities)

	v1.GET("/localities/:uf", getLocalitiesByUF)

	return router
}
