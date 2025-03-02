# Changelog

## v0.5.1 (March 1, 2025)

### New Features

* Introduce support for shutting down; more configurable ready funcs (8318dce)

### Bug fixes and improvements

* Bug fix: Address `TraceLevel` parsing; verbose usage (c7d3c20)
* Chores:
    * Update dependent versions; go1.23 (1ca1feb, 26813dc, 68dcd55, 7767928, e38c734)
    * Update GitHub configuration: Dependabot (74ddc90)
    * Update GoReleaser configuration (f929992)
    * Use newer mechanism in tool configuration (1e84bc6)
    * Improve linting; address lint errors (2446f7d)
    * Update documentation (e268fde)

## v0.5.0 (June 8, 2023)

### New Features
* Select bind address when connecting with `wig` (21498ba)
* Expression evaluation on redirects (`redirect.*` expressions) (d70a124)
* Encapsulate `SetURLValue` arg (701dfd3)
* Add pattern support to file downloader (803ee26)
* Add support for random to `ExpandGlobals` (4bd4013)
* Add support for time to `ExpandGlobals` (ee3c518)
* Whitespace handling in exprs (aa14540)
* Add TLS support to server (34f074a)
* Transport middleware (722a09e)
* `ExpandURL` (3ba483f)

### Bug fixes and improvements

* Fallback Expr evaluation generally; in redirects (2376317)
* Remove conversion to string within response Expander (588e0cd)
* Add context to `Downloader.OpenContext` (8d06f81)
* Add request URI to URL expansions (c4bd91e)
* Encapsulate http.Client on Client interface (e869eb2)
* Refactor Client to handle responses and output (ac37f9c)
* TraceLogger fixes (134e108)
* Bug fix: Print out full request URI when tracing requests (112b7d1)
* Bug fix: Trace template (783798d)
* Bug fix: regression - not writing out request headers (7ef2d08)
* Bug fix: Use `statusCode` instead of `status` expr (ab1f111)
* Print actual server listener bind address (6679344)
* Simplify `Response.CopyHeadersTo` (ea9dc03)
* Extract Prefix expanders (67f27b5)
* Chores:
    * Fix releaser: Ensure unique artifact archives names (8caab8b)

## v0.4.0 (May 15, 2023)

### New Features

* Add support for `Options` to `VirtualPath` (f30ce84)
* Trace out response header (26a4191)
* Support meta expr format to support built-in access log formats (a9c0509)
* Allow string slices in `WithHeader` (4d853b4)
* Server header middleware; `--server` option (27ef2bd)
* Support fail fast mode (f67d7b0)
* Support expression toggling between writers (850d9a4)
* Support color variables within write-out expressions (53104d2)
* Expand global variables (5f8bde7)
* `StripComponents` (acd1c0e)
* Introduce download middleware (4b318d2)
* Support compiling expressions; access log (1524efc)
* Server middleware (2412dbd)
* Ping handler (0fcdf2a)
* `HandlerSpec` (ba59818)
* Encapsulate file server handler (9e60373)
* Redirect handler (508c3ae)

### Bug fixes and improvements

* Request logger format refactor (c8218ee)
* Support additional interrupts on server (9c61f83)
* Update documentation httpserver (b9474fd)
* Chores:
    * Update dependent versions (e766cf9, a7292dc)
    * GitHub CI configuration (8975b61)
    * Goreleaser refactoring (c43c7db)
    * Addresses issues of style, deprecation warnings (a3429bd)
    * Makefile: collapse output from go generate (2a5040f)

## v0.3.2 (March 14, 2023)

### New Feature

* Allow header names in write out expr (7bb1314)

### Bug fixes and improvements

* Bug fix: requests in virtual paths are rooted (bf27ca9)
* Additional tests for write out expressions (7bb1314)
* Chore: GitHub configuration (4c3d77b)

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
