package monitor

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"zhub/internal/zsub"
)

func init() {
	// 1.日志文件 定期分割归档

}

func StartWatch() {

	r := gin.Default()

	r.Group("/users")

	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, zsub.Info())
	})
	r.GET("/cleanup", func(c *gin.Context) {
		zsub.Hub.Clearup()
		c.JSON(http.StatusOK, "+OK")
	})
	r.GET("/retimer", func(c *gin.Context) {
		zsub.Hub.ReloadTimer()
		c.JSON(http.StatusOK, "+reload timer ok")
	})
	r.GET("/topic/publish", func(c *gin.Context) {
		topic := c.Query("topic")
		value := c.Query("value")

		zsub.Hub.Publish(topic, value)
		c.JSON(http.StatusOK, "+OK")
	})
	r.GET("/topic/delay", func(c *gin.Context) {
		topic := c.Query("topic")
		value := c.Query("value")
		delay := c.Query("delay")

		zsub.Hub.Delay([]string{"delay", topic, value, delay})
		c.JSON(http.StatusOK, "+OK")
	})

	watchAddr := zsub.Conf.Service.Watch
	r.Run(watchAddr)
}
