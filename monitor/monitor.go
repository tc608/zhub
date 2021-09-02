package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"zhub/zsub"
)

func StartHttp() {
	dir, _ := os.Getwd()
	webDir := path.Join(dir, "/public")

	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/info", info)
	http.HandleFunc("/cleanup", cleanup)
	http.HandleFunc("/retimer", retimer)
	http.HandleFunc("/topic/publish", publish)

	http.ListenAndServe(":1217", nil)
}

func publish(w http.ResponseWriter, r *http.Request) {
	topic := r.FormValue("topic")
	value := r.FormValue("value")
	zsub.ZSubx().Publish(topic, value)
	renderJson(w, "+ok")
}

func retimer(w http.ResponseWriter, r *http.Request) {
	zsub.ZSubx().ReloadTimer()
	renderJson(w, "+reload timer ok")
}

func cleanup(w http.ResponseWriter, r *http.Request) {
	zsub.ZSubx().Clearup()
	renderJson(w, "+OK")
}

func info(w http.ResponseWriter, r *http.Request) {
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
}
