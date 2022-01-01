# Windermere
Windermere is an open source SCIM server for the EGIL profile of SS12000:2018.

It is built to make it easy to quickly get started with EGIL as a service
provider.

## Quick start

Before getting started you need a few things:

 * A host with the Go compiler installed
 * URL and public keys to the authentication federation
 * A certificate to use in PEM format
 * A configuration file for Windermere

You can find the URL and public keys for Kontosynk at [Kontosynk](https://www.skolfederation.se/teknisk-information/kontosynk/tekniska-miljoer/)

A certificate can be generated with OpenSSL, for instance:

```
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes
```

Put the certificates and keys file in a new directory, and create a Windermere
config file, for instance named config.yaml.

You can start with the following example:

```
MetadataURL: https://fed.skolfederation.se/trial/md/kontosynk.jws
JWKSPath: /home/joe/windermere/jwks.trial
MetadataCachePath: /home/joe/windermere/metadata-cache.json
Cert: /home/joe/windermere/cert.pem
Key: /home/joe/windermere/key.pem
ListenAddress: :443
EnableLimiting: true
LimitRequestsPerSecond: 10
LimitBurst: 20
StorageType: sqlite
StorageSource: storage.db
```

Replace path names as appropriate.

The StorageType specifies with SQL driver to use. Currently included drivers
are:

 * sqlite (SQLite)
 * sqlserver (Microsoft SQL Server)

Depending on which driver is used, the StorageSource has a different format
for specifying access to the database. For SQLite you can simply specify a
filename.

Example for SQL Server:

```
sqlserver://<user>:<passwd>@<host>?database=<database>
```

### Building

To build Windermere, go into the directory `cmd/windermere` and run `go build`,
this should give you an executable named `windermere` in the same directory.

### Running

Go to the directory where you have the certificates and windermere configuration
and run windermere, for instance:

```
$ windermere config.yaml
```

Specify the full path to the binary unless you've added it to your `$PATH`.
