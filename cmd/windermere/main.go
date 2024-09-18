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
	"github.com/kardianos/service"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// Windermere can be run in interactive mode as a regular executable,
// or as a service. The code below is used when using the program as
// a service.

const serviceName = "windermere"
const serviceDescription = "Windermere EGIL SCIM Server"

// This implements service.Interface (in the kardianos/service package)
type serviceInterface struct {
	// The program will wait for signals on this channel before shutting down.
	// When running as a regular interactive program, we will connect SIGINT
	// and SIGTERM to this channel. When running as a service, the Stop function
	// below will be called by the service package and manually send a signal
	// to this channel.
	signals chan os.Signal

	// When running as a service, the process will send to this channel to signal
	// when proper shutdown is complete.
	done chan bool
}

func (si serviceInterface) Start(s service.Service) error {
	go run(si.signals, si.done)
	return nil
}

func (si serviceInterface) Stop(s service.Service) error {
	// When running as a service, the service package takes care of
	// signals, so we'll emulate a SIGINT sent to the process.
	si.signals <- syscall.SIGINT
	<-si.done
	return nil
}

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
func waitForShutdownSignal(signals chan os.Signal) {
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
	CNFLogFilePath            = "LogPath"
	CNFSkolsynkListenAddress  = "SkolsynkListenAddress"
	CNFSkolsynkAuthHeader     = "SkolsynkAuthHeader"
	CNFSkolsynkCert           = "SkolsynkCert"
	CNFSkolsynkKey            = "SkolsynkKey"
	CNFSkolsynkClients        = "SkolsynkClients"
	CNFSS12000v2ListenAddress = "SS12000v2ListenAddress"
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
		client, ok := arr[i].(map[string]interface{})
		if !ok {
			return nil, err
		}

		getString := func(m map[string]interface{}, s string) (string, error) {
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

// This is like the programs core main function. The actual main()
// will take care of parsing arguments and behaves a bit differently
// depending on whether we're running interactively, as a service,
// or if we're installing/uninstalling a service.
func run(signals chan os.Signal, done chan bool) {
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
		handler = Limiter(handler, tenantGetter,
			rate.Limit(viper.GetFloat64(CNFLimitRequestsPerSecond)),
			viper.GetInt(CNFLimitBurst))
	}

	beTimeout := configuredSeconds(CNFBackendTimeout)
	if beTimeout >= 1*time.Second {
		handler = PanicReportTimeoutHandler(handler, beTimeout, "Backend timeout")
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
			Handler: server.AuthMiddleware(handler, mdstore, nil),

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

	// Possibly create the SS12000 v2 HTTP server
	if viper.IsSet(CNFSS12000v2ListenAddress) {
		ss12000v2Server := &http.Server{
			Handler: NewSS12000v2TenantMux(wind),
			Addr:    viper.GetString(CNFSS12000v2ListenAddress),

			ReadHeaderTimeout: configuredSeconds(CNFReadHeaderTimeout),
			ReadTimeout:       configuredSeconds(CNFReadTimeout),
			WriteTimeout:      configuredSeconds(CNFWriteTimeout),
			IdleTimeout:       configuredSeconds(CNFIdleTimeout),
		}
		// TODO: Needs a proper goroutine which can be asked to quit?
		go func() {
			err := ss12000v2Server.ListenAndServeTLS(certFile, keyFile)

			if err != http.ErrServerClosed {
				log.Fatalf("Unexpected SS12000v2 server exit: %s", err.Error())
			}
		}()
	}

	waitForShutdownSignal(signals)

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

	if done != nil {
		done <- true
	}
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

	// Parse command line
	var install = flag.Bool("install", false, "install as a service")
	var uninstall = flag.Bool("uninstall", false, "uninstall as a service")
	var serviceUser = flag.String("user", "", "user to run the service")
	var servicePassword = flag.String("password", "", "password for service user")

	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Missing configuration file path")
	}

	configPath := flag.Arg(0)

	viper.SetConfigFile(configPath)

	must(viper.ReadInConfig())

	if viper.IsSet(CNFLogFilePath) {
		f, err := os.OpenFile(viper.GetString(CNFLogFilePath), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	verifyRequired(CNFCert, CNFKey)

	if viper.IsSet(CNFListenAddress) {
		verifyRequired(CNFJWKSPath, CNFMDCachePath,
			CNFMDEntityID, CNFMDBaseURI, CNFMDOrganization, CNFMDOrganizationID)
	} else if viper.IsSet(CNFSkolsynkListenAddress) {
		verifyRequired(CNFSkolsynkClients)
	} else {
		log.Fatalf("No listen address configured (configure at least one of %s or %s", CNFListenAddress, CNFSkolsynkListenAddress)
	}

	// The main goroutine will wait for signals on this channel before
	// terminating.
	sigs := make(chan os.Signal)

	// Extra (system-dependent) service options
	opts := make(service.KeyValue)

	if *servicePassword != "" {
		opts["Password"] = *servicePassword
	}

	// Set up the service, not necessarily used if we're running
	// the program as a regular interactive service.
	serviceConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceDescription,
		Arguments:   flag.Args(),
		UserName:    *serviceUser,
		Option:      opts,
	}
	si := &serviceInterface{signals: sigs, done: make(chan bool)}
	s, err := service.New(si, serviceConfig)
	if err != nil {
		log.Fatalf("Cannot create the service: %s", err.Error())
	}

	// Run the program in different ways depending on whether we're
	// being called from a service manager or not, and whether we're
	// trying to install/uninstall the program as a service.
	if service.Interactive() {
		if *install {
			err = s.Install()
			if err != nil {
				log.Fatalf("Cannot install the service: %s", err.Error())
			}
		} else if *uninstall {
			err = s.Uninstall()
			if err != nil {
				log.Fatalf("Cannot uninstall the service: %s", err.Error())
			}
		} else {
			// Don't use the service, just run directly so Ctrl-C works
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			run(sigs, nil)
		}
	} else {
		err = s.Run()
		if err != nil {
			log.Fatalf("Cannot start the service: %s", err.Error())
		}
	}
}
