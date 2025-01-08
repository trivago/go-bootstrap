# Changelog

## [1.2.3](https://github.com/trivago/go-bootstrap/compare/v1.2.2...v1.2.3) (2025-01-08)


### Bug Fixes

* reload certificate when file has changed ([0b8572e](https://github.com/trivago/go-bootstrap/commit/0b8572e8840d4955ef5315cb8010f5ae30d0ea49))


### Miscellaneous

* add nix lock file ([79f3ac4](https://github.com/trivago/go-bootstrap/commit/79f3ac455d2d6da67f4aa9dbd9e4bb178755a0e8))
* add pre-commit hooks ([b818237](https://github.com/trivago/go-bootstrap/commit/b818237fc25e3e680e78002d30e16c5970686d8a))
* **deps:** bump golang.org/x/net from 0.19.0 to 0.23.0 ([#9](https://github.com/trivago/go-bootstrap/issues/9)) ([0e1dc2f](https://github.com/trivago/go-bootstrap/commit/0e1dc2f4020ca70fdac81d3b104f615fd32811b3))

## [1.2.2](https://github.com/trivago/go-bootstrap/compare/v1.2.1...v1.2.2) (2024-03-28)


### Miscellaneous

* **deps:** bump golang.org/x/crypto from 0.16.0 to 0.17.0 ([#6](https://github.com/trivago/go-bootstrap/issues/6)) ([6939604](https://github.com/trivago/go-bootstrap/commit/6939604851f70564e7927d7f8442218869e66ce6))
* **deps:** bump google.golang.org/protobuf from 1.31.0 to 1.33.0 ([#7](https://github.com/trivago/go-bootstrap/issues/7)) ([ddb331a](https://github.com/trivago/go-bootstrap/commit/ddb331ae5637b99a7c14f9133aa949989767e68f))

## [1.2.1](https://github.com/trivago/go-bootstrap/compare/v1.2.0...v1.2.1) (2023-12-01)


### Bug Fixes

* crash on 2nd call to GetCertificate ([a589200](https://github.com/trivago/go-bootstrap/commit/a58920038421ce278f45abc59dc0b4d1bb49f725))
* verify client cert before returning it ([9879a80](https://github.com/trivago/go-bootstrap/commit/9879a805635c3fb06376a8e33365521231ee81e0))


### Miscellaneous

* add localhost certificate for testing ([335d504](https://github.com/trivago/go-bootstrap/commit/335d504c204a6ee16f57b34e2072c1e4787e52b2))

## [1.2.0](https://github.com/trivago/go-bootstrap/compare/v1.1.1...v1.2.0) (2023-12-01)


### Features

* let httpserver.New return errors ([9d48bec](https://github.com/trivago/go-bootstrap/commit/9d48bec95c8a61baab064a8a9bdd13c4e19b450e))


### Bug Fixes

* make cert reloading thread safe ([f717cf9](https://github.com/trivago/go-bootstrap/commit/f717cf968e8717b567d72ef021ccb6d63883c3ed))


### Miscellaneous

* fix go build action ([cebf956](https://github.com/trivago/go-bootstrap/commit/cebf9564ec2db5f575935052b7f16962494afa9f))

## [1.1.1](https://github.com/trivago/go-bootstrap/compare/v1.1.0...v1.1.1) (2023-12-01)


### Bug Fixes

* update dependencies ([60b8780](https://github.com/trivago/go-bootstrap/commit/60b878006bbea5c187c330f355b3981b78549310))


### Miscellaneous

* add codeowners ([19051ea](https://github.com/trivago/go-bootstrap/commit/19051eae7d875148276d0d8b6f59f8dda5fac074))

## [1.1.0](https://github.com/trivago/go-bootstrap/compare/v1.0.0...v1.1.0) (2023-12-01)


### Features

* add TLS support to httpserver ([474c5d1](https://github.com/trivago/go-bootstrap/commit/474c5d18c8b5899c03cbcd952d1978fb9a9ca211))
