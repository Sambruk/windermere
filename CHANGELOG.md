## Releases

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
