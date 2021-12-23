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

func verifyRequired(keys ...string) {
	for _, key := range keys {
		if !viper.IsSet(key) {
			log.Fatalf("Missing required configuration setting: %s", key)
		}
	}
}

func configuredSeconds(setting string) time.Duration {
	return time.Duration(viper.GetInt(setting)) * time.Second
}

func waitForShutdownSignal() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals
}

func main() {
	viper.SetDefault("MetadataURL", "https://fed.skolfederation.se/prod/md/kontosynk.jws")
	viper.SetDefault("MetadataDefaultCacheTTL", 3600)
	viper.SetDefault("MetadataNetworkRetry", 60)
	viper.SetDefault("MetadataBadContentRetry", 3600)
	viper.SetDefault("ReadHeaderTimeout", 5)
	viper.SetDefault("ReadTimeout", 20)
	viper.SetDefault("WriteTimeout", 40)
	viper.SetDefault("IdleTimeout", 60)
	viper.SetDefault("BackendTimeout", 30)
	viper.SetDefault("EnableLimiting", false)
	viper.SetDefault("LimitRequestsPerSecond", 10.0)
	viper.SetDefault("LimitBurst", 50)
	viper.SetDefault("StorageType", "file")
	viper.SetDefault("StorageSource", "SS12000.json")

	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Missing configuration file path")
	}

	configPath := flag.Arg(0)

	viper.SetConfigFile(configPath)

	must(viper.ReadInConfig())

	verifyRequired("JWKSPath", "MetadataCachePath", "Cert", "Key", "ListenAddress")

	mdstore := fedtls.NewMetadataStore(
		viper.GetString("MetadataURL"),
		viper.GetString("JWKSPath"),
		viper.GetString("MetadataCachePath"),
		fedtls.DefaultCacheTTL(configuredSeconds("MetadataDefaultCacheTTL")),
		fedtls.NetworkRetry(configuredSeconds("MetadataNetworkRetry")),
		fedtls.BadContentRetry(configuredSeconds("MetadataBadContentRetry")))

	certFile := viper.GetString("Cert")
	keyFile := viper.GetString("Key")

	mdTLSConfigManager, err := server.NewMetadataTLSConfigManager(certFile, keyFile, mdstore)

	if err != nil {
		log.Fatalf("Failed to create TLS configuration: %v", err)
	}

	tenantGetter := func(c context.Context) string {
		return server.EntityIDFromContext(c)
	}

	wind, err := windermere.New(viper.GetString("StorageType"), viper.GetString("StorageSource"), tenantGetter)
	var handler http.Handler
	handler = wind

	if err != nil {
		log.Fatalf("Failed to initialize Windermere: %v", err)
	}

	enableLimiting := viper.GetBool("EnableLimiting")

	if enableLimiting {
		handler = server.Limiter(handler,
			rate.Limit(viper.GetFloat64("LimitRequestsPerSecond")),
			viper.GetInt("LimitBurst"))
	}

	beTimeout := configuredSeconds("BackendTimeout")
	if beTimeout >= 1*time.Second {
		handler = http.TimeoutHandler(handler, beTimeout, "Backend timeout")
	}

	srv := &http.Server{
		// Wrap the HTTP handler with authentication middleware.
		Handler: server.AuthMiddleware(handler, mdstore),

		// In order to use the authentication middleware, the server needs
		// to have a ConnContext configured so the middleware can access
		// connection specific information.
		ConnContext: server.ContextModifier(),

		ReadHeaderTimeout: configuredSeconds("ReadHeaderTimeout"),
		ReadTimeout:       configuredSeconds("ReadTimeout"),
		WriteTimeout:      configuredSeconds("WriteTimeout"),
		IdleTimeout:       configuredSeconds("IdleTimeout"),
	}

	// Set up a TLS listener with certificate authorities loaded from
	// federation metadata (and dynamically updated as metadata gets refreshed).
	address := viper.GetString("ListenAddress")
	listener, err := tls.Listen("tcp", address, mdTLSConfigManager.Config())

	if err != nil {
		log.Fatalf("Failed to listen to %s (%v)", address, err)
	}

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

	log.Printf("Server closed, waiting for metadata store to close...")
	mdstore.Quit()

	log.Printf("Done.")
}
