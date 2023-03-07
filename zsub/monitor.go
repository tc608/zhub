package zsub

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
)

func init() {
	// 1.日志文件 定期分割归档

}

func StartWatch() {
	dir, _ := os.Getwd()
	webDir := path.Join(dir, "/public")

	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/info", info)
	http.HandleFunc("/cleanup", cleanup)
	http.HandleFunc("/retimer", retimer)
	http.HandleFunc("/topic/publish", publish)

	watchAddr := GetStr("service.zhub.watch", "0.0.0.0:1217")
	log.Println("zhub.watch = ", watchAddr)
	http.ListenAndServe(watchAddr, nil)
}

func publish(w http.ResponseWriter, r *http.Request) {
	topic := r.FormValue("topic")
	value := r.FormValue("value")
	zsub.Publish(topic, value)
	renderJson(w, "+ok")
}

// retimer 重载定时调度
func retimer(w http.ResponseWriter, r *http.Request) {
	zsub.ReloadTimer()
	renderJson(w, "+reload timer ok")
}

func cleanup(w http.ResponseWriter, r *http.Request) {
	zsub.Clearup()
	renderJson(w, "+OK")
}

func info(w http.ResponseWriter, r *http.Request) {
	info := Info()
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
}
