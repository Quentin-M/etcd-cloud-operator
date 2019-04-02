# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

-   [aws] Ignition Config is uploaded and fetched via S3 and stored encrypted using KMS
-   [aws] Backups are now encrypted using KMS
-   Kubernetes Support #26
-   Adds support for arbitary ignition config #28
-   [aws] Security group is part of generated outputs

### Changed

-   Node exporter pinned to v0.17.0 #35
-   [aws] ASG default cooldown changed to `0` #35
-   Generated certs have a validity of 5y now #34

## [3.3.3b] - 2018-04-20

### Fixed

-   Fix restoration issue in Docker

## [3.3.3] - 2018-04-07

### Added

-   Introduce functional testing

## [3.3] - 2018-02-07

### Added

-   Update to etcd 3.3
