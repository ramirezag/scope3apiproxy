# Scope3 API Proxy

Project that proxies [Scope3 APIs](https://docs.scope3.com/reference/measure-1) to
provide [cross-cutting features](#cross-cutting-feature-implemented).

## Cross-cutting feature implemented

### Caching

Caching is implemented based on the known requirements given. The following describes how and uses emissions API as an
example.

- **In-Memory** - The application has built-in memory for caching. This is suitable to achieve <= 10ms API response.
- **Cache Aside (Lazy Loading)** - On API call to emissions API, fetches the emission given the properties from the cache.
  If any of the properties is not in the cache, they are fetched from the Scope3 API server.
- **Eviction policy** - When cache capacity is reached, the app evicts record in the cache based on the following conditions
  checked in order:
    - **Priority** - Records are compared against the priority number optionally provided by the client in the emissions
      API. Higher number means higher priority
    - **Frequency** - Records are compared against how often the record is queried. Least frequently used (LFU) are evicted
      first.
    - **TTL** - Records that are about to expire are evicted first.

# Setup the project for local development

### Prerequisite

- [Go 1.23](https://go.dev/doc/install) is installed
- [GIT](https://git-scm.com/downloads) is installed

### Clone the project for development

```
$ git clone https://github.com/ramirezag/scope3apiproxy
$ cd scope3apiproxy
$ go mod download
```

**Note:** Some IDEs like [Goland](https://www.jetbrains.com/help/go/import-project-or-module-wizard.html) streamlines some go development operations like `go mod download`.

# How to run the app

## How to run the project locally

There are 2 ways to run the app:

- Using default config and overriding only the api key
  through [environment variable](#application-configs) -
  `SCOPE3_APIKEY=<your_api_key> go run main.go`
- Copying the default config then run the app

  ```shell
    $ cp config.json config.local.json # then modify
    $ go run main.go
  ```

## How to build and run a production ready application

```
$ go build -ldflags "-s -w -extldflags '-static' -X 'main.version=1.0.0' -X 'main.commitHash=$(git rev-parse HEAD)'" -o scope3apiproxy
$ export ENVIRONMENT=production
$ export SCOPE3_APIKEY=<your_api_key>
$ ./scope3apiproxy
```

**Note:** GOOS=<os> GOARCH=<arch> may be needed if the machine where the app is built on is different from the target
deployment server. You can find the full list of the
values [here](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63).

## Application Configs

All [application config](./config.json) used by the app can be overridden through the environment variable using the
upper-cased combination of the property names. Nested properties must be separated by underscore.

| Config                           | Environment variable             | Description                                                                                                                                                                               |
|:---------------------------------|:---------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| port                             | PORT                             | Port used by the app. Defaults to 8080                                                                                                                                                    |
| gracefulShutdownTimeoutInSeconds | GRACEFULSHUTDOWNTIMEOUTINSECONDS | How many seconds the app will wait for pending process (eg, request) running in the app before it shutsdown                                                                               |
| scope3.host                      | SCOPE3_HOST                      | Host of the scope3 API server. Should start with http or https. Defaults to [https://api.scope3.com](https://docs.scope3.com/reference)                                                   |
| scope3.apiKey                    | SCOPE3_APIKEY                    | API key allowed to make a call to scope3 API server.                                                                                                                                      |
| scope3.timeoutInSeconds          | SCOPE3_TIMEOUTINSECONDS          | Time before the request to scope3 API server is interrupted - see [Timeout time.Duration in http#Client](https://pkg.go.dev/net/http#Client). Defaults to 10s.                            |
| scope3.maxIdleConnections        | SCOPE3_MAXIDLECONNECTIONS        | Max idle connections with scope3 API server - see [MaxIdleConns in http#Transport](https://pkg.go.dev/net/http#Transport). Defaults to 10.                                                |
| scope3.idleConnTimeoutInSeconds  | SCOPE3_IDLECONNTIMEOUTINSECONDS  | Maximum amount of time an idle (keep-alive) connection will remain idle before closing - see [IdleConnTimeout in http#Transport](https://pkg.go.dev/net/http#Transport). Defaults to 30s. |
| cache.capacity                   | CACHE_CAPACITY                   | Maximum capacity of the cache. Defaults to 1000                                                                                                                                           |
| cache.emissionTtlInMinutes       | CACHE_EMISSIONTTLINMINUTES       | TTL of an emission record to stay in the cache. Defaults to 60 minutes.                                                                                                                   |

# How to test the app

The request body structure of the `emissions API` is implemented similar to the expected structure of [measure API of scope3](https://docs.scope3.com/reference/measure-1).
But only `inventoryId`, `impressions`, and `utcDatetime` are marked as required.

```shell
curl -X POST "http://localhost:8080/api/v1/emissions" \
--header 'content-type: application/json' \
--data '{"rows": [
{"country": "US","channel": "web","inventoryId":"nytimes.com","impressions":1000,"utcDatetime":"2024-10-31"}
]}'
```

Additionally, each row in the request body can have `priority` so that client/customers can have control of how the API caches the data.

```shell
curl -X POST "http://localhost:8080/api/v1/emissions" \
--header 'content-type: application/json' \
--data '{"rows": [
{"country": "US","channel": "web","inventoryId":"nytimes.com","impressions":1000,"utcDatetime":"2024-10-31","priority":10}
]}'
```

The response would be something like

```json
{
  "data": {
    "nytimes.com": {
      "adSelection": {},
      "compensated": {},
      ... other fields ...
    }
  }
}
```