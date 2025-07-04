## Releases

## 1.3.0 (2025-07-04)
  - New "dummy" backend with no memory (for testing purposes) (#46)

## v1.2.3 (2025-01-07)
  - Bowness upgraded to v1.1.4
    This was due to a security alert by GitHub dependabot. The security
    issue doesn't seem to affect Bowness or Windermere (see bowness repository
    for details).

## v1.2.2 (2024-09-30)
  - Golang upgraded to v1.22 (#42)

## v1.2.1 (2024-09-19)
  - Re-usable and extendable main package (#39)

## v1.2.0 (2024-09-16)
#### Features
  - Windermere can run as a Windows service (#18)

## v1.1.0 (2024-09-09)
  - The User type now supports securityMarking (#36)
    This attribute is not supported by the SQL backend at the moment.

## v1.0.2 (2024-02-28)
  Upgrades dependencies due to security alerts (GitHub dependabot).

## v1.0.1 (2023-02-28)
  Upgrades dependencies due to security alerts (GitHub dependabot).

## v1.0.0 (2023-02-22)
  Version 1.0.0 set to signal that Windermere is considered mature enough
  for production deployment. Also to show intent to maintain backwards
  compatibility (until v2.0), according to semantic versioning.

  The only difference between previous version (v0.8.0) and v1.0.0 is updated
  dependencies due to security alerts (GitHub dependabot). It doesn't seem
  like the security alerts were relevant for Windermere.

## v0.8.0 (2023-01-16)
  - The StudentGroup type now supports schoolType (#30)
    This attribute is not supported by the SQL backend at the moment.
  - The Activity type now supports parentActivity (#30)
    This attribute is not supported by the SQL backend at the moment.

## v0.7.0 (2022-11-29)
  - The User type now supports userRelations (#28)
    userRelations are not supported by the SQL backend at the moment.

## v0.6.1 (2022-06-15)
  - Fixed parsing of Skolsynk clients after Viper upgrade (#25)

## v0.6.0 (2022-05-31)
### New features
  - Accepts PUT towards a resource type end point (#23)

## v0.5.2 (2022-05-17)
#### Bugfixes
  - Better stack traces when the timeout handler is used (#19)
  - Rate limiting now also works for API-key based authentication (#20)

## v0.5.1 (2022-03-23)
#### Bugfixes
  - If the in-memory backend was used, parsed objects weren't deleted
    properly (#16)

## v0.5.0 (2022-03-18)
#### New features
  - Support for API-key based authentication (Skolsynk) (#11)
  - We will now retry DB connection if it fails at startup (#13)
