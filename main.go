//go:build go1.20
// +build go1.20

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// -------------- 数据模型 --------------
type Agent struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	LastSeen int64  `json:"last_seen"`
	Status   string `json:"status"`
}

type TEMSReport struct {
	TEMSName  string  `json:"tems_name"`
	Timestamp int64   `json:"timestamp"`
	Agents    []Agent `json:"agents"`
}

// -------------- 内存存储 --------------
var store struct {
	sync.RWMutex
	TEMS map[string][]Agent
}

func init() {
	store.TEMS = make(map[string][]Agent)
}

// -------------- 接收端点 --------------
func reportHandler(w http.ResponseWriter, r *http.Request) {
	var rep TEMSReport
	if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	store.Lock()
	store.TEMS[rep.TEMSName] = rep.Agents
	store.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// -------------- 查询 API --------------
func apiHandler(w http.ResponseWriter, r *http.Request) {
	store.RLock()
	defer store.RUnlock()
	_ = json.NewEncoder(w).Encode(store.TEMS)
}

// -------------- 主函数 --------------
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/teps/report", reportHandler).Methods("POST")
	r.HandleFunc("/api", apiHandler)
	// 静态文件（index.html / topo.html）
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./web"))))

	log.Println("GEPS mock listening http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
