## Releases

## Unreleased
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
