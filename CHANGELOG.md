# Changelog

## v0.3.1 (March 13, 2023)

### Bug fixes and improvements

* Improved error handling in `DownloadMode`, `File` adapter (29422d1)
* Updates to GitHub actions (83da39c)
* Update engineering platform (b051f3a)
* Unit tests for internal URI templates (3c01ddd)

## v0.3.0 (March 12, 2023)

### New Features

* Introduce `Location`, an indirection of location resolver (015e06e)
* `Middleware` (140e279)
    * `RequestID` middleware (2c874ea)
    * `WithHeaders` middleware (185dce1)
    * Extract `WithHeader` middleware (d0a08cb)
* `Integrity` (c54f891)
* `NewDownloaderTo`, slight `Downloader` refactoring (2a08239)
* `VirtualPath` (753e3ab)
* Write-out expressions (12f190e)
* Make `HideDirectoryListing` API (8ca9e95)
* Partially expand URI templates, and commands (77ae8f6)
* Convert between body content type (2a7ed8a)
* `QueryString` values (b58979d)
* Add support for env var `INSECURE_SKIP_VERIFY` (095cbdb)
* Split `CopyTo` into `CopyHeadersTo` (ad0c624)
* Introduce `--no-output` flag (759f068)
* Add `SetMaxHeaderBytes` (abd3a7c)
* Introduce `Handle` to add to server handler mux (19ba51e)

### Bug fixes and improvements

* Basic tests for `FetchAndPrint`, testable transport (d69e36c)
* Route `FetchAndPrint` output to context `Stdout` (9a66533)
* Rename `Auth` to `Authenticator` method (6d7d55f)
* Bug fix: ensure directories exist when writing download (f09e853)
* Rename `Include{ => Response}Headers` (3d91e28)
* Extract `ContextValue`, and documentation (eafcb45)
* Update user agent string for base (a3eabf1)
* Bug fix: Safer handling of `*Context` in auth middleware (c78db69)
* Fix `IdleTimeout` flags registration (abd3a7c)
* Chores:
    * Upgrade Ginkgo versions (085f7bf)
    * Unit tests covering more action methods (3d91e28)
    * Fixes for various issues of style (a1724fe)
    * Unit tests for various flags (3ce65c4)
    * Updates to documentation; typos (bbdc2be)
    * Update CI to latest Go versions (6f5e7d4)
    * Update dependent versions, including Go (c2f619f)
    * Update dependent versions (dbfc355)
    * Regenerate radstubs (53aeddd)

## v0.2.0 (September 5, 2022)

### New Features

* Support additional client APIs and commands for Next protos (69c6f5e), server name (e6ed732), time (a3e375c), TLS 1.3 curves (29d2aa2)
* Support client certs and CAs (c4a6462)
* Open in browser (9eab4aa)
* Fill form value, JSON (initial implementation) (ae3267d)
* Vars as a flag (8367e24)

### Bug fixes and improvements

* Allow case insensitive HTTP method names (357b84f)
* Relocate commands into internal package (204fed3)
* Allow marshal to be used in content type (a871e48)
* Provide Copy convention for various flag values (ed656a7)
* Increase code coverage (28bd053, 172dbf3, f0d88e0)
* Bug fix: Auth flag missing description (8367e24)
* Chores:
  * Update Goreleaser pipeline (01485f9)
  * Makefile: Introduce coveragereport, coverage (5b79e71)
  * Add Go1.19 to build (22386df)
  * Update dependent versions (b4fc19b)

## v0.1.0 (July 9, 2022)

* Initial version :sunrise:
