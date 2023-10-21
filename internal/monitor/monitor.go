package monitor

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"zhub/internal/zbus"
)

var r = gin.Default()

func init() {
	// 1.日志文件 定期分割归档

}

func StartWatch() {

	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.GET("/_/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, zbus.Info())
	})
	r.GET("/_/cleanup", func(c *gin.Context) {
		zbus.Bus.Clearup()
		c.JSON(http.StatusOK, "+OK")
	})

	r.GET("/timer/reload", func(c *gin.Context) {
		zbus.Bus.ReloadTimer()
		c.JSON(http.StatusOK, "+reload timer ok")
	})
	r.GET("/topic/publish", func(c *gin.Context) {
		topic := c.Query("topic")
		value := c.Query("value")

		zbus.Bus.Publish(topic, value)
		c.JSON(http.StatusOK, "+OK")
	})
	r.GET("/topic/delay", func(c *gin.Context) {
		topic := c.Query("topic")
		value := c.Query("value")
		delay := c.Query("delay")

		zbus.Bus.Delay([]string{"delay", topic, value, delay})
		c.JSON(http.StatusOK, "+OK")
	})

	// reload the auth configuration
	r.GET("/auth/reload", func(c *gin.Context) {
		zbus.AuthManager.Reload()
		c.JSON(http.StatusOK, "+OK")
	})

	watchAddr := zbus.Conf.Service.Watch
	r.Run(watchAddr)
}
