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
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	CNFAdminListenAddress     = "AdminListenAddress"
	CNFMDEntityID             = "MetadataEntityID"
	CNFMDBaseURI              = "MetadataBaseURI"
	CNFMDOrganization         = "MetadataOrganization"
	CNFMDOrganizationID       = "MetadataOrganizationID"
	CNFValidateUUID           = "ValidateUUID"
	CNFValidateSchoolUnitCode = "ValidateSchoolUnitCode"
	CNFSkolsynkListenAddress  = "SkolsynkListenAddress"
	CNFSkolsynkAuthHeader     = "SkolsynkAuthHeader"
	CNFSkolsynkCert           = "SkolsynkCert"
	CNFSkolsynkKey            = "SkolsynkKey"
	CNFSkolsynkClients        = "SkolsynkClients"
)

// Parses the config value for clients to a map[string]string
// (from tenant name to API key)
func parseClients(value interface{}) (map[string]string, error) {
	res := make(map[string]string)
	err := errors.New("invalid clients specification")

	arr, ok := value.([]interface{})
	if !ok {
		return nil, err
	}

	for i := range arr {
		client, ok := arr[i].(map[interface{}]interface{})
		if !ok {
			return nil, err
		}

		getString := func(m map[interface{}]interface{}, s string) (string, error) {
			val, ok := m[s]
			if !ok {
				return "", err
			}
			res, ok := val.(string)
			if !ok {
				return "", err
			}
			return res, nil
		}
		name, e := getString(client, "name")
		if e != nil {
			return nil, e
		}
		key, e := getString(client, "key")
		if e != nil {
			return nil, e
		}

		res[name] = key
	}
	return res, nil
}

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
		CNFAdminListenAddress:     "",
		CNFValidateUUID:           true,
		CNFValidateSchoolUnitCode: true,
		CNFSkolsynkAuthHeader:     "X-API-Key",
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

	verifyRequired(CNFCert, CNFKey)

	if viper.IsSet(CNFListenAddress) {
		verifyRequired(CNFJWKSPath, CNFMDCachePath,
			CNFMDEntityID, CNFMDBaseURI, CNFMDOrganization, CNFMDOrganizationID)
	} else if viper.IsSet(CNFSkolsynkListenAddress) {
		verifyRequired(CNFSkolsynkClients)
	} else {
		log.Fatalf("No listen address configured (configure at least one of %s or %s", CNFListenAddress, CNFSkolsynkListenAddress)
	}

	certFile := viper.GetString(CNFCert)
	keyFile := viper.GetString(CNFKey)

	// Windermere needs a function to get the currently authenticated
	// SCIM tenant from the current Context.
	tenantGetter := func(c context.Context) string {
		tenant := APIKeyAuthenticatedTenantFromContext(c)
		if tenant != nil {
			return *tenant
		}
		return server.EntityIDFromContext(c)
	}

	// Configurable validation of SS12000 objects
	validator := windermere.CreateOptionalValidator(
		viper.GetBool(CNFValidateUUID),
		viper.GetBool(CNFValidateSchoolUnitCode),
	)

	// Create the Windermere SCIM handler
	wind, err := windermere.New(viper.GetString(CNFStorageType), viper.GetString(CNFStorageSource), tenantGetter, validator)

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

	var fedtlsServer *http.Server
	var mdstore *fedtls.MetadataStore
	// Possibly setup EGIL server with Federated TLS authentication
	if viper.IsSet(CNFListenAddress) {
		// Setup federated TLS metadata store
		mdstore = fedtls.NewMetadataStore(
			viper.GetString(CNFMDURL),
			viper.GetString(CNFJWKSPath),
			viper.GetString(CNFMDCachePath),
			fedtls.DefaultCacheTTL(configuredSeconds(CNFMDDefaultCacheTTL)),
			fedtls.NetworkRetry(configuredSeconds(CNFMDNetworkRetry)),
			fedtls.BadContentRetry(configuredSeconds(CNFMDBadContentRetry)))

		// The TLS config manager is used by the tls.Listener below to configure
		// TLS according to the TLS federation.
		mdTLSConfigManager, err := server.NewMetadataTLSConfigManager(certFile, keyFile, mdstore)

		if err != nil {
			log.Fatalf("Failed to create TLS configuration: %v", err)
		}

		// Create the HTTP server
		fedtlsServer = &http.Server{
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

		// Start the main HTTP server
		go func() {
			err := fedtlsServer.Serve(listener)

			if err != http.ErrServerClosed {
				log.Fatalf("Unexpected server exit: %v", err)
			}
		}()
	}

	var skolsynkServer *http.Server
	// Possibly start Skolsynk HTTP server (API-key based authentication)
	if viper.IsSet(CNFSkolsynkListenAddress) {
		clients, err := parseClients(viper.Get(CNFSkolsynkClients))

		if err != nil {
			log.Fatalf("Failed to parse clients from config: %v", err)
		}

		// Create the HTTP server
		skolsynkServer = &http.Server{
			// Wrap the HTTP handler with authentication middleware.
			Handler: APIKeyAuthMiddleware(handler,
				viper.GetString(CNFSkolsynkAuthHeader),
				clients),
			Addr: viper.GetString(CNFSkolsynkListenAddress),

			ReadHeaderTimeout: configuredSeconds(CNFReadHeaderTimeout),
			ReadTimeout:       configuredSeconds(CNFReadTimeout),
			WriteTimeout:      configuredSeconds(CNFWriteTimeout),
			IdleTimeout:       configuredSeconds(CNFIdleTimeout),
		}

		skolsynkCertFile := certFile
		skolsynkKeyFile := keyFile

		if viper.IsSet(CNFSkolsynkCert) {
			skolsynkCertFile = viper.GetString(CNFSkolsynkCert)
		}
		if viper.IsSet(CNFSkolsynkKey) {
			skolsynkKeyFile = viper.GetString(CNFSkolsynkKey)
		}

		go func() {
			err := skolsynkServer.ListenAndServeTLS(skolsynkCertFile, skolsynkKeyFile)

			if err != http.ErrServerClosed {
				log.Fatalf("Unexpected Skolsynk server exit: %v", err)
			}
		}()
	}

	// Possibly start the admin HTTP server
	adminAddress := viper.GetString(CNFAdminListenAddress)
	if adminAddress != "" {
		http.Handle("/metadata", metadataHandler(certFile,
			viper.GetString(CNFMDEntityID), viper.GetString(CNFMDBaseURI),
			viper.GetString(CNFMDOrganization), viper.GetString(CNFMDOrganizationID)))
		go func() {
			log.Println(http.ListenAndServeTLS(adminAddress, certFile, keyFile, nil))
		}()
	}

	waitForShutdownSignal()

	log.Printf("Shutting down, waiting for active requests to finish...")

	if fedtlsServer != nil {
		err = fedtlsServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("Failed to gracefully shutdown federated TLS server: %v", err)
		}
	}

	if skolsynkServer != nil {
		err = skolsynkServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("Failed to gracefully shutdown Skolsynk server: %v", err)
		}
	}

	err = wind.Shutdown()
	if err != nil {
		log.Printf("Failed to gracefully shutdown Windermere: %v", err)
	}

	if mdstore != nil {
		log.Printf("Server closed, waiting for metadata store to close...")
		mdstore.Quit()
	}

	log.Printf("Done.")
}
