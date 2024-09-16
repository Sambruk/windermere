# Windermere
Windermere is an open source SCIM server for the EGIL profile of SS12000:2018.

It is built to make it easy to quickly get started with EGIL as a service
provider.

## Quick start

Before getting started you need a few things:

 * A host with the Go compiler installed (v1.16 or later)
 * URL and public keys to the authentication federation
 * A certificate to use in PEM format
 * A configuration file for Windermere

You can find the URL and public keys for Moa (previously Kontosynk) at [Skolfederation's wiki](https://www.skolfederation.se/teknisk-information/moa/tekniska-miljoer/)

A certificate can be generated with OpenSSL, for instance:

```
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes
```

Put the certificates and keys file in a new directory, and create a Windermere
config file, for instance named config.yaml.

You can start with the following example:

```
# Configuration of the authentication federation and metadata
MetadataURL: https://fed.skolfederation.se/trial/md/kontosynk.jws
JWKSPath: /home/joe/windermere/jwks.trial
MetadataCachePath: /home/joe/windermere/metadata-cache.json

# Information to be published about this server in the
# federation metadata.
MetadataEntityID: https://egil.serviceprovider.com
MetadataBaseURI: https://egil.serviceprovider.com
MetadataOrganization: Service Provider Ltd.
MetadataOrganizationID: SE1234567890

# HTTP server settings
Cert: /home/joe/windermere/cert.pem
Key: /home/joe/windermere/key.pem
ListenAddress: :443

# Rate limiting
EnableLimiting: true
LimitRequestsPerSecond: 20
LimitBurst: 20

# Storage backend
StorageType: sqlite
StorageSource: storage.db
```

Replace path names, and information about your organization as appropriate.

The StorageType specifies which SQL driver to use. Currently included drivers
are:

 * sqlite (SQLite)
 * sqlserver (Microsoft SQL Server)
 * mysql (MySQL, MariaDB, Percona Server, Google CloudSQL or Sphinx)

Depending on which driver is used, the StorageSource has a different format
for specifying access to the database. For SQLite you can simply specify a
filename.

Example for SQL Server:

```
sqlserver://<user>:<passwd>@<host>?database=<database>
```

Example for MySQL/MariaDB:

```
<user>:<password>@<host>/<database>?multiStatements=true
```

`multiStatements=true` is currently needed for this driver
(other drivers allow this by default).

### Building

To build Windermere, go into the directory `cmd/windermere` and run `go build`,
this should give you an executable named `windermere` in the same directory.

If you don't want to build yourself you can find binaries on GitHub (under Releases).

### Running

Go to the directory where you have the certificates and windermere configuration
and run windermere, for instance:

```
$ windermere config.yaml
```

Specify the full path to the binary unless you've added it to your `$PATH`.

## Metadata to upload to the federation operator

Before clients can connect to the server you need to upload your metadata
to the federation operator (Moa). Windermere can generate your
metadata for you, but you need to activate the administration HTTP interface.

### Administration HTTP interface

You can activate the administration interface by specifying the `AdminListenAddress`
parameter in the configuration file, for instance:

```
AdminListenAddress: 127.0.0.1:4443
```

:warning: **Please note**: the administration interface has no authentication
and is not meant to be publicly exposed. Make sure the address cannot be reached
except by your own staff.

Currently the administration interface only implements two end-points:

 * Metadata (`/metadata`)
 * Debug tools (`/debug/pprof`)

You can download the metadata with your web browser, or for instance with curl:

```
curl -k https://127.0.0.1:4443/metadata
```

The administration interface uses the same certificate as the EGIL SCIM server
(hence the need for `-k` above in case the certificate is self signed).

## Access log

If you want to log all (authenticated) requests to the server you can specify
a path to an access log file, for instance:

```
AccessLogPath: /home/windermere/access.log
```

The access log will show who made the request (ip and federation entity id),
what URL was requested, result code and execution time for each request.

:warning: **Please note**: the access log can grow quickly since each request
is logged, so if you want to have it switched on permanently in production
you may need to set up frequent log rotation (for instance with the standard
Unix logrotate tool).

## Skolsynk (or other client with API-key authentication)

If you wish to allow Skolsynk to connect you need to configure a separate
listening address, for instance:

```
SkolsynkListenAddress: :4430
```

If you wish you can configure which HTTP header to use for authentication key:

```
SkolsynkAuthHeader: X-API-Key
```

The default, if you don't configure this, is X-API-Key.

Then you also need to create API keys for your clients:

```
SkolsynkClients:
  - name: skolsynkGoogle
    key: HfACARee4G3qrtBx
  - name: skolsynkMicrosoft
    key: fjEGgDnDBAaBh4rN
```

What you specify after `name` will be used as the tenant name in the database
(similar to entity ID if you're using Federated TLS (Moa)). You should generate
the `key` yourself and hand it over to Skolsynk in a secure way.

By default, the same certificate will be used for the HTTPS traffic for Skolsynk,
but if you wish you can specify a separate certificate also:

```
SkolsynkCert: /home/windermere/skolsynkcert.pem
SkolsynkKey: /home/windermere/skolsynkkey.pem
```

You can choose to run only Federated TLS (Moa), only Skolsynk, or both at the same
time (by configuring `ListenAddress` and/or `SkolsynkListenAddress`).

## Running as a service

Windermere can run as a regular command line program, or as a service.

For Linux, there's an example systemd service definition [here](configs/windermere.service).

For Windows, the service can be installed by running the program with `-install`:

```
windermere -install c:\windermere\config.yaml
```

You can remove the service with the `-uninstall` flag.

If you wish you can also specify user and password for the service:

```
windermere -install -user ben -password verysecret c:\windermere\config.yaml
```

Note that the path to the config file should be absolute when you install the
service since the process might not start from the same directory when running
as a service. You may also want to make sure the paths inside the config file
are absolute for the same reason.

If you don't want to specify password on the command line you can configure that
in the Windows Service Manager instead.

### Logging

If you're running Windermere as a service you may want to send logging to a file
(depending on what your operating system does with standard output and standard
error for services). Use the `LogPath` setting in your config file:

```
LogPath: c:/windermere/log.txt
```
Note that if you use this when installing or uninstalling the service, any error
messages from installing or uninstalling will also end up in the log file.
