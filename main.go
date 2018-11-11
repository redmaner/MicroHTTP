package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const version = "v0.12"

// Main function
func main() {

	initLogger("MicroHTTP-")

	args := os.Args
	if len(args) == 1 {
		showHelp()
	}

	// Handle arguments
	// To start MicroHTTP you need to define the path to the main configuration file
	if _, err := os.Stat(args[1]); err == nil {
		var mCfg microConfig
		loadConfigFromFile(args[1], &mCfg)
		if valid, err := validateConfig(args[1], &mCfg); valid && err == nil {
			startServer(&mCfg)
		} else {
			logAction(logERROR, err)
			os.Exit(1)
		}

	} else {
		showHelp()
	}
}

// Function to start Server
func startServer(mCfg *microConfig) {

	// Set micro struct
	m := micro{
		config: *mCfg,
		vhosts: make(map[string]microConfig),
		md: metricsData{
			enabled: mCfg.Metrics.Enabled,
			paths:   make(map[int]map[string]int),
		},
	}

	// If virtual hosting is enabled, all the configurations of the vhosts are loaded
	if m.config.Serve.VirtualHosting {
		for k, v := range m.config.Serve.VirtualHosts {
			var cfg microConfig
			loadConfigFromFile(v, &cfg)
			if valid, err := validateConfigVhost(v, &cfg); !valid || err != nil {
				logAction(logERROR, err)
				os.Exit(1)
			}
			m.vhosts[k] = cfg
		}
	}

	// Determine the router of MicroHTTP
	// The router is the default multiplexer of the net/http package
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.httpServe)
	if m.config.Metrics.Enabled {
		mux.HandleFunc(m.config.Metrics.Path+"/", m.httpMetrics)
	}

	// If TLS is enabled the server will start in TLS
	if m.config.TLS && httpCheckTLS(&m.config) {
		logAction(logNONE, fmt.Errorf("MicroHTTP is listening on port %s with TLS", mCfg.Port))
		tlsc := httpCreateTLSConfig()
		ms := http.Server{
			Addr:      mCfg.Address + ":" + mCfg.Port,
			Handler:   mux,
			TLSConfig: tlsc,
		}

		// This is meant to listen for signals. A signal will stop MicroHTTP
		done := make(chan bool)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		go func() {
			<-quit
			logAction(logNONE, fmt.Errorf("Server is shutting down..."))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ms.SetKeepAlivesEnabled(false)
			if err := ms.Shutdown(ctx); err != nil {
				logAction(logNONE, fmt.Errorf("Could not gracefully shutdown the server: %v\n", err))
			}
			close(done)
		}()

		// Start the server
		err := ms.ListenAndServeTLS(mCfg.TLSCert, mCfg.TLSKey)
		if err != nil && err != http.ErrServerClosed {
			logAction(logERROR, fmt.Errorf("Starting server failed: %s", err))
			return
		}

		<-done
		logAction(logNONE, fmt.Errorf("MicroHTTP stopped"))

		// IF TLS is disabled the server is started without TLS
		// Never run non TLS servers in production!
	} else {
		logAction(logDEBUG, fmt.Errorf("MicroHTTP is listening on port %s", mCfg.Port))
		http.ListenAndServe(mCfg.Address+":"+mCfg.Port, mux)
	}
}

// Function to show help
func showHelp() {
	fmt.Printf("MicroHTTP version %s\n\nUsage: microhttp </path/to/config.json>\n\n", version)
	os.Exit(1)
}
