# goapm

[![Go Report Card](https://goreportcard.com/badge/github.com/hedon954/goapm)](https://goreportcard.com/report/github.com/hedon954/goapm)
[![codecov](https://codecov.io/github/hedon954/goapm/graph/badge.svg?token=FEW1EL1FKG)](https://codecov.io/github/hedon954/goapm)
[![CI](https://github.com/hedon954/goapm/workflows/build/badge.svg)](https://github.com/hedon954/goapm/actions)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/hedon954/goapm?sort=semver)](https://github.com/hedon954/goapm/releases)

`goapm` is a toolkit for monitoring and observability of golang applications. It provides a set of libraries that are wrapped around `opentelemetry`.

## Example
- [goapm-example](https://github.com/hedon954/goapm-example)


## Features
- [x] Components support `opentelemetry`
  - [x] sql.DB
  - [x] gorm.DB
  - [x] RedisV6
  - [x] RedisV9
  - [x] HTTP
  - [x] Gin
  - [x] GRPC Server
  - [x] GRPC Client
- [x] Metrics
- [x] AutoPProf
- [x] APM
- [x] RotateLog


## Architecture
![architecture](https://hedonspace.oss-cn-beijing.aliyuncs.com/img/image-20241031195051363.png)



