package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/t3kpunk/wolweb/arptable"
	// "github.com/irai/arp"
	// "github.com/yaklang/yaklang/common/utils/arptable"
)

// Global variables
var appConfig AppConfig
var appData AppData

func main() {

	setWorkingDir()
	loadConfig()
	loadData()
	initArpTable()
	updateArpTable()
	monitorArpTable()
	setupWebServer()
}

func setWorkingDir() {

	thisApp, err := os.Executable()
	if err != nil {
		log.Fatalf("Error determining the directory. \"%s\"", err)
	}
	appPath := filepath.Dir(thisApp)
	os.Chdir(appPath)
	log.Printf("Set working directory: %s", appPath)

}

func loadConfig() {

	err := cleanenv.ReadConfig("config.json", &appConfig)
	if err != nil {
		log.Fatalf("Error loading config.json file. \"%s\"", err)
	}
	log.Printf("Application configuratrion loaded from config.json")

}

func setupWebServer() {

	// Init HTTP Router - mux
	router := mux.NewRouter()

	// Define base path. Keep it empty when VDir is just "/" to avoid redirect loops
	// Add trailing slash if basePath is not empty
	basePath := ""
	if appConfig.VDir != "/" {
		basePath = appConfig.VDir
		router.HandleFunc(basePath, redirectToHomePage).Methods("GET")
	}

	// map directory to server static files
	router.PathPrefix(basePath + "/static/").Handler(http.StripPrefix(basePath+"/static/", CacheControlWrapper(http.FileServer(http.Dir("./static")))))

	// Define Home Route
	router.HandleFunc(basePath+"/", renderHomePage).Methods("GET")

	// Define Wakeup functions with a Device Name
	router.HandleFunc(basePath+"/wake/{deviceName}", wakeUpWithDeviceName).Methods("GET")
	router.HandleFunc(basePath+"/wake/{deviceName}/", wakeUpWithDeviceName).Methods("GET")

	// Define Data save Api function
	router.HandleFunc(basePath+"/data/save", saveData).Methods("POST")

	// Define Data get Api function
	router.HandleFunc(basePath+"/data/get", getData).Methods("GET")

	// Define health check function
	router.HandleFunc(basePath+"/health", checkHealth).Methods("GET")

	// Setup Webserver
	httpListen := net.ParseIP(appConfig.Host).String() + ":" + strconv.Itoa(appConfig.Port)
	log.Printf("Startup Webserver on \"%s\"", httpListen)

	srv := &http.Server{
		Handler: gziphandler.GzipHandler(handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(router)),
		Addr:    httpListen,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func CacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=31536000")
		h.ServeHTTP(w, r)
	})
}

func initArpTable() {
	arptable.FlushARP()
	for _, c := range appData.Devices {
		_, err := arpingHost(c.Name)
		if err == nil {
			Ping(c.Name, 1)
		}
	}
}

func inArpTable(mac string) string {
	arpTable := arptable.Table()
	for idx := range arpTable {
		if arpTable[idx] == mac {
			return Online
		}
	}
	return Offline
}

func checkAlive(dev Device) string {
	alive := inArpTable(dev.Mac)
	if alive == Online {
		return Online
	}
	if alive == Offline {
		_, err := net.LookupHost(dev.Name)
		if err != nil {
			return Death
		}
	}
	return Offline
}

func monitorArpTable() {
	go func() {
		// Infinite scannning devices
		for {
			mutex := &sync.Mutex{}
			mutex.Lock()
			// NOTE: Dynamic slice. Devices can be added or removed,
			var lenght = len(appData.Devices)
			var dev = make([]Device, lenght)

			for idx, c := range appData.Devices {
				dev[idx] = Device{Name: c.Name, Mac: c.Mac, BroadcastIP: c.BroadcastIP, Alive: checkAlive(c)}
			}
			appData.Devices = dev
			mutex.Unlock()
			time.Sleep(1200 * time.Millisecond)
		}
	}()
}

func updateArpTable() {
	go func() {
		// Infinite scannning devices
		for {
			mutex := &sync.Mutex{}
			mutex.Lock()
			var lenght = len(appData.Devices)
			var dev = make([]Device, lenght)

			for idx, c := range appData.Devices {
				dev[idx] = Device{Name: c.Name, Mac: c.Mac, BroadcastIP: c.BroadcastIP, Alive: c.Alive}
				if c.Alive == Offline {
					status, err := arpingHost(c.Name)
					if err != nil {
						log.Printf("arp pinging device: %s", status)
						if Ping(c.Name, 10) {
							log.Printf("device '%s' is reachable via broadcast ping", c.Name)
						}

					}
				}
			}
			appData.Devices = dev
			mutex.Unlock()
			time.Sleep(12000 * time.Millisecond)
		}
	}()
}
