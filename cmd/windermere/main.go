/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2021 Föreningen Sambruk
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.

 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.

 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sambruk/windermere/windermere"
	"github.com/joesiltberg/bowness/fedtls"
	"github.com/joesiltberg/bowness/server"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

func must(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

// Verifies that certain parameters are set in the configuration file
func verifyRequired(keys ...string) {
	for _, key := range keys {
		if !viper.IsSet(key) {
			log.Fatalf("Missing required configuration setting: %s", key)
		}
	}
}

// Convenience method for getting a configuration parameter that
// specifies a duration in seconds.
func configuredSeconds(setting string) time.Duration {
	return time.Duration(viper.GetInt(setting)) * time.Second
}

// Blocks until we get a signal to shut down
func waitForShutdownSignal() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals
}

// Configuration parameters
const (
	CNFMDURL                  = "MetadataURL"
	CNFMDDefaultCacheTTL      = "MetadataDefaultCacheTTL"
	CNFMDNetworkRetry         = "MetadataNetworkRetry"
	CNFMDBadContentRetry      = "MetadataBadContentRetry"
	CNFMDCachePath            = "MetadataCachePath"
	CNFReadHeaderTimeout      = "ReadHeaderTimeout"
	CNFReadTimeout            = "ReadTimeout"
	CNFWriteTimeout           = "WriteTimeout"
	CNFIdleTimeout            = "IdleTimeout"
	CNFBackendTimeout         = "BackendTimeout"
	CNFEnableLimiting         = "EnableLimiting"
	CNFLimitRequestsPerSecond = "LimitRequestsPerSecond"
	CNFLimitBurst             = "LimitBurst"
	CNFStorageType            = "StorageType"
	CNFStorageSource          = "StorageSource"
	CNFAccessLogPath          = "AccessLogPath"
	CNFJWKSPath               = "JWKSPath"
	CNFCert                   = "Cert"
	CNFKey                    = "Key"
	CNFListenAddress          = "ListenAddress"
)

func main() {
	// Configuration defaults
	defaults := map[string]interface{}{
		CNFMDURL:                  "https://fed.skolfederation.se/prod/md/kontosynk.jws",
		CNFMDDefaultCacheTTL:      3600,
		CNFMDNetworkRetry:         60,
		CNFMDBadContentRetry:      3600,
		CNFReadHeaderTimeout:      5,
		CNFReadTimeout:            20,
		CNFWriteTimeout:           40,
		CNFIdleTimeout:            60,
		CNFBackendTimeout:         30,
		CNFEnableLimiting:         false,
		CNFLimitRequestsPerSecond: 10.0,
		CNFLimitBurst:             50,
		CNFStorageType:            "file",
		CNFStorageSource:          "SS12000.json",
		CNFAccessLogPath:          "",
	}
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Missing configuration file path")
	}

	configPath := flag.Arg(0)

	viper.SetConfigFile(configPath)

	must(viper.ReadInConfig())

	verifyRequired(CNFJWKSPath, CNFMDCachePath, CNFCert, CNFKey, CNFListenAddress)

	// Setup federated TLS metadata store
	mdstore := fedtls.NewMetadataStore(
		viper.GetString(CNFMDURL),
		viper.GetString(CNFJWKSPath),
		viper.GetString(CNFMDCachePath),
		fedtls.DefaultCacheTTL(configuredSeconds(CNFMDDefaultCacheTTL)),
		fedtls.NetworkRetry(configuredSeconds(CNFMDNetworkRetry)),
		fedtls.BadContentRetry(configuredSeconds(CNFMDBadContentRetry)))

	certFile := viper.GetString(CNFCert)
	keyFile := viper.GetString(CNFKey)

	// The TLS config manager is used by the tls.Listener below to configure
	// TLS according to the TLS federation.
	mdTLSConfigManager, err := server.NewMetadataTLSConfigManager(certFile, keyFile, mdstore)

	if err != nil {
		log.Fatalf("Failed to create TLS configuration: %v", err)
	}

	// Windermere needs a function to get the currently authenticated
	// SCIM tenant from the current Context.
	tenantGetter := func(c context.Context) string {
		return server.NormalizedEntityIDFromContext(c)
	}

	// Create the Windermere SCIM handler
	wind, err := windermere.New(viper.GetString(CNFStorageType), viper.GetString(CNFStorageSource), tenantGetter)

	if err != nil {
		log.Fatalf("Failed to initialize Windermere: %v", err)
	}

	// Setup various middlware handlers between Windermere and the http.Server
	var handler http.Handler
	handler = wind

	enableLimiting := viper.GetBool(CNFEnableLimiting)

	if enableLimiting {
		handler = server.Limiter(handler,
			rate.Limit(viper.GetFloat64(CNFLimitRequestsPerSecond)),
			viper.GetInt(CNFLimitBurst))
	}

	beTimeout := configuredSeconds(CNFBackendTimeout)
	if beTimeout >= 1*time.Second {
		handler = http.TimeoutHandler(handler, beTimeout, "Backend timeout")
	}

	accessLogPath := viper.GetString(CNFAccessLogPath)
	if accessLogPath != "" {
		handler = accessLogHandler(handler, accessLogPath, tenantGetter)
	}

	// Create the HTTP server
	srv := &http.Server{
		// Wrap the HTTP handler with authentication middleware.
		Handler: server.AuthMiddleware(handler, mdstore),

		// In order to use the authentication middleware, the server needs
		// to have a ConnContext configured so the middleware can access
		// connection specific information.
		ConnContext: server.ContextModifier(),

		ReadHeaderTimeout: configuredSeconds(CNFReadHeaderTimeout),
		ReadTimeout:       configuredSeconds(CNFReadTimeout),
		WriteTimeout:      configuredSeconds(CNFWriteTimeout),
		IdleTimeout:       configuredSeconds(CNFIdleTimeout),
	}

	// Set up a TLS listener with certificate authorities loaded from
	// federation metadata (and dynamically updated as metadata gets refreshed).
	address := viper.GetString(CNFListenAddress)
	listener, err := tls.Listen("tcp", address, mdTLSConfigManager.Config())

	if err != nil {
		log.Fatalf("Failed to listen to %s (%v)", address, err)
	}

	// Start the HTTP server
	go func() {
		err := srv.Serve(listener)

		if err != http.ErrServerClosed {
			log.Fatalf("Unexpected server exit: %v", err)
		}
	}()

	waitForShutdownSignal()

	log.Printf("Shutting down, waiting for active requests to finish...")

	err = srv.Shutdown(context.Background())
	if err != nil {
		log.Printf("Failed to gracefully shutdown server: %v", err)
	}

	err = wind.Shutdown()
	if err != nil {
		log.Printf("Failed to gracefully shutdown Windermere: %v", err)
	}

	log.Printf("Server closed, waiting for metadata store to close...")
	mdstore.Quit()

	log.Printf("Done.")
}
