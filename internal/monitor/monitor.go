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

	watchAddr := zsub.Conf.Service.Watch
	r.Run(watchAddr)

	/*dir, _ := os.Getwd()
	webDir := path.Join(dir, "/public")

	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/info", info)
	http.HandleFunc("/cleanup", cleanup)
	http.HandleFunc("/retimer", retimer)
	http.HandleFunc("/topic/publish", publish)

	watchAddr := zsub.Conf.Service.Watch
	log.Println("zhub.watch = ", watchAddr)
	http.ListenAndServe(watchAddr, nil)*/
}

/*func publish(w http.ResponseWriter, r *http.Request) {
	topic := r.FormValue("topic")
	value := r.FormValue("value")
	zsub.Hub.Publish(topic, value)
	renderJson(w, "+ok")
}

// retimer 重载定时调度
func retimer(w http.ResponseWriter, _ *http.Request) {
	zsub.Hub.ReloadTimer()
	renderJson(w, "+reload timer ok")
}

func cleanup(w http.ResponseWriter, _ *http.Request) {
	zsub.Hub.Clearup()
	renderJson(w, "+OK")
}

func info(w http.ResponseWriter, _ *http.Request) {
	info := zsub.Info()
	renderJson(w, info)
}

func renderJson(w http.ResponseWriter, d interface{}) {
	var bytes []byte

	if str, ok := d.(string); ok {
		bytes = []byte(str)
	} else {
		bytes, _ = json.Marshal(d)
		w.Header().Set("content-type", "application/json; charset=utf-8;")
	}
	w.Write(bytes)
}*/
