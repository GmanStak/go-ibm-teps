//go:build go1.20
// +build go1.20

package main

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

type Config struct {
	ListenAddr string `yaml:"listen_addr"`
	Basic      struct {
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
	} `yaml:"basic"`
}

var cfg Config

func loadConfig(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Fatalf("parse yaml: %v", err)
	}
}

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

func basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := cfg.Basic.User
		pass := cfg.Basic.Pass
		if user == "" && pass == "" {
			next.ServeHTTP(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="GEPS"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
	loadConfig("config.yaml") // 读取 config.yaml

	r := mux.NewRouter()
	// 1. 公开端点：TEMS 推送
	r.HandleFunc("/teps/report", reportHandler).Methods("POST")

	// 2. 受保护端点
	protected := r.PathPrefix("/").Subrouter()
	protected.HandleFunc("/api", apiHandler)
	protected.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./web"))))
	protected.Use(basicAuth)

	addr := cfg.ListenAddr
	if addr == "" {
		addr = ":8081"
	}
	log.Printf("GEPS mock listening http://localhost%s  (basic auth enabled)", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
