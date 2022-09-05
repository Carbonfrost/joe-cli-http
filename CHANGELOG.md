# Changelog

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
