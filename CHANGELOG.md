# Changelog

---
## [0.0.1] - 2024-10-31

Completed the base features of APM, wrapping common tools like Redis, MySQL, HTTP, Gin, and Gorm with OpenTelemetry, providing tracing, metrics, and logging capabilities.

### ⚙️ Miscellaneous Chores

- push example protos - ([ec12785](https://github.com/hedon954/goapm/commit/ec127851ee54c1cc7456b43875c34324e49fc091)) - hedon954
- push goapm - ([0a0bd8e](https://github.com/hedon954/goapm/commit/0a0bd8ee1c68a6ed273844015af4301aebbc2b37)) - hedon954
- remove exmaples folder - ([9b7544b](https://github.com/hedon954/goapm/commit/9b7544bb40e978c6d93aa99b51487e2ff2a99d4d)) - hedon954

### ⛰️ Features

- **(apm)** finsih goapm api - ([9c4adc2](https://github.com/hedon954/goapm/commit/9c4adc22273b6979135899c0c68a572e31b2b43e)) - hedon954
- **(ci)** upgrade golangci-lint in github action workflow - ([07b74de](https://github.com/hedon954/goapm/commit/07b74deca03608209515a2352d4baf6f0290a7cf)) - hedon954
- **(gin)** wrap gin with otel - ([09c3551](https://github.com/hedon954/goapm/commit/09c3551dc5c689579676070629d42ae2ad342f11)) - hedon954
- **(gorm)** provider gorm otel wrapper and add unit tests for it - ([9f156af](https://github.com/hedon954/goapm/commit/9f156afd61fafbf6277350a258db5c6df3a5c4da)) - hedon954
- **(grpc)** wrapper otel with grpc server and grpc server, and upgrade go version to 1.23.2 - ([8f4d6f9](https://github.com/hedon954/goapm/commit/8f4d6f99d98ff9b11bde5b764b8c866a05d07ca7)) - hedon954
- **(grpc)** support set options when create new grpc server and client - ([f62341c](https://github.com/hedon954/goapm/commit/f62341c76ee0b620e14b4db2deaf4939b4922ddf)) - hedon954
- **(http)** wrap http with otel - ([af6d447](https://github.com/hedon954/goapm/commit/af6d447f0df269c0f07425ee7bf8218b42a50e3d)) - hedon954
- **(redis)** wrap redis with otel - ([63920c3](https://github.com/hedon954/goapm/commit/63920c34dc6c5eec8a4c4125775b3fb1fb319d11)) - hedon954
- **(sql)** wrap sql.DB with otel trace, log and metric - ([1dfe87e](https://github.com/hedon954/goapm/commit/1dfe87ef59a7ff80435e689268c2857cf407b5d2)) - hedon954
- add db_utils - ([163ae45](https://github.com/hedon954/goapm/commit/163ae453e80301e1bf20e4a90064b11b581a139e)) - hedon954
- fix go mod error - ([6ad99cc](https://github.com/hedon954/goapm/commit/6ad99cca21815bc8af604b50032d98044b714363)) - hedon954
- http support metrics and heartbeat api default - ([2da750e](https://github.com/hedon954/goapm/commit/2da750e8f0c562512ca89ae7ecd0d6a77e13987c)) - hedon954

### 📚 Documentation

- **(readme)** add architecture - ([6c70acb](https://github.com/hedon954/goapm/commit/6c70acbbcf00106a78291ecb0114e7a77270b221)) - hedon954
- **(readme)** add badges - ([d54a1ec](https://github.com/hedon954/goapm/commit/d54a1ecc314d194239df87fc9b01cdb716c7aa57)) - hedon954
- update readme - ([ed8cb97](https://github.com/hedon954/goapm/commit/ed8cb97a7f2f17b34b31ff291d501c9293e54b2f)) - hedon954

### 🚜 Refactor

- **(grpc)** expose Close method for grpcServer - ([4c61ae1](https://github.com/hedon954/goapm/commit/4c61ae1b67d633ace02aa87bb8d20f197fc67fde)) - hedon954
- **(grpc)** support to dynamic get grpc server address - ([6585414](https://github.com/hedon954/goapm/commit/6585414a60bc17dcac423d04a003b069fdc93897)) - hedon954
- check duplicated mysql, gorm and redis register - ([5f7bde4](https://github.com/hedon954/goapm/commit/5f7bde47699c3c4d0832efd203fa1b5c158158da)) - hedon954
- use `error` to replace `haserror` for jaeger ui show - ([8735900](https://github.com/hedon954/goapm/commit/8735900a9452d47e0a91ff652a09b8e19744a22e)) - hedon954

### 🧪 Tests

- **(sql)** add unit tests for mysql wrapper driver - ([7a41351](https://github.com/hedon954/goapm/commit/7a4135161db373c0f339e7311b0994eaab73534b)) - hedon954

<!-- generated by git-cliff -->
