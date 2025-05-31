# Configuration options

PixivFE can be configured using environment variables or loading from a config file.

The order of precedence is as follows (from highest to lowest):

1. Environment variables
2. Configuration file
3. Default values

## Configuration file

By default, PixivFE tries to load the configuration from a `config.yml` file in the current working directory, if it exists.
You can create this file and put any of the configuration options below in it.

Here is an example of a minimal configuration file:

```yaml
basic:
  token:
    - 123456_arstdhnei
```

For all available configuration options, see the [example config.yml](https://gitlab.com/pixivfe/PixivFE/-/blob/v3/deploy/config.yml) in the repository.

You can specify a different path for the configuration file using the `-config` option.
The supplied config file must be in YAML format. Example:

```bash
# Build PixivFE and execute
go build
./pixivfe -config deploy/config.yml

# You can do `./pixivfe -h` to get help
```

It is also possible to use an environment variable to specify the configuration file:

```bash
# Build PixivFE and execute
go build
PIXIVFE_CONFIGFILE=deploy/config.yml ./pixivfe
```

## Environment variables

Any configuration option can be set as an environment variable. If a configuration option was both set in configuration file and/or in an environment variable, PixivFE will prioritize using environment variables.

For example:

```bash
PIXIVFE_TOKEN=YOUR_PIXIV_COOKIE PIXIVFE_PORT=8282 ./pixivfe
```

<!-- TODO: the following should link to v2 once v3 is merged -->

For all available configuration options, check out the [.env.example](https://gitlab.com/pixivfe/PixivFE/-/blob/v3/deploy/.env.example) in the repository.

## Basic options

**These options must be nested under a `basic:` block in `config.yml`.**

### `PIXIVFE_CONFIGFILE`

| YAML name | Environment variable | Required | Default        | Options |
| --------- | -------------------- | -------- | -------------- | ------- |
| -         | `PIXIVFE_CONFIGFILE` | No       | `./config.yml` | -       |

Specifies path to a YAML configuration file. This environment variable takes priority over the `-config` command line flag.

### `PIXIVFE_HOST`

| YAML name | Environment variable | Required | Default     | Options                |
| --------- | -------------------- | -------- | ----------- | ---------------------- |
| `host`    | `PIXIVFE_HOST`       | No       | `localhost` | Hostname or IP address |

The hostname or IP address that PixivFE should listen on and accept incoming connections from.

Mutually exclusive with `PIXIVFE_UNIXSOCKET`.

!!!note
    If you're **not using a reverse proxy** or **running PixivFE inside Docker**, you should set `PIXIVFE_HOST=0.0.0.0`. This will allow PixivFE to accept connections from any IP address or hostname. If you don't set this, PixivFE will refuse direct connections from other machines or devices on your network.

### `PIXIVFE_PORT`

| YAML name | Environment variable | Required | Default | Options     |
| --------- | -------------------- | -------- | ------- | ----------- |
| `port`    | `PIXIVFE_PORT`       | No       | `8282`  | Port number |

The port number that PixivFE should listen on and accept incoming connections from.

Mutually exclusive with `PIXIVFE_UNIXSOCKET`.

### `PIXIVFE_UNIXSOCKET`

| YAML name    | Environment variable | Required | Default | Options                                                              |
| ------------ | -------------------- | -------- | ------- | -------------------------------------------------------------------- |
| `unixSocket` | `PIXIVFE_UNIXSOCKET` | No       | -       | [UNIX socket path](https://en.wikipedia.org/wiki/Unix_domain_socket) |

The UNIX socket path that PixivFE should use.

Mutually exclusive with `PIXIVFE_HOST`/`PIXIVFE_PORT`.

### `PIXIVFE_TOKEN`

| YAML name | Environment variable | Required | Default | Options                                   |
| --------- | -------------------- | -------- | ------- | ----------------------------------------- |
| `token`   | `PIXIVFE_TOKEN`      | Yes      | -       | Comma-separated tokens (multiple allowed) |

`PHPSESSID` cookie(s) for authentication with the pixiv API.

Multiple tokens can be specified.

Environment variable format:

```sh
PIXIVFE_TOKEN=token1,token2,token3
```

Config file format:

```yaml
token:
  - token1
  - token2
```

### `PIXIVFE_TZ`

| YAML name | Environment variable | Required | Default   | Options                                                                          |
| --------- | -------------------- | -------- | --------- | -------------------------------------------------------------------------------- |
| `tz`      | `PIXIVFE_TZ`         | No       | `Etc/Utc` | [tz database name](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) |

The timezone setting used to properly handle pixiv API endpoints that work using local time. This affects features like the rankings page and comments.

Set this to match the timezone of the IP address you're using to make pixiv API requests.

## Content proxy servers

**These options must be nested under a `contentProxies:` block in `config.yml`.**

pixiv requires `Referer: https://www.pixiv.net/` in the HTTP request headers to fetch images from their servers.

If any of these individual settings is not set, the built-in proxy will be used for that specific service.

For setting up your own proxy server, refer to [hosting an image proxy server](image-proxy-server.md). Alternatively, see the [list of public image proxies](../public-image-proxies.md) if you prefer to use an existing deployment.

### `PIXIVFE_IMAGEPROXY`

| YAML name    | Environment variable | Required | Default          | Options |
| ------------ | -------------------- | -------- | ---------------- | ------- |
| `imageProxy` | `PIXIVFE_IMAGEPROXY` | No       | (built-in proxy) | URL     |

The URL of a server that acts as a reverse proxy for i.pximg.net.

### `PIXIVFE_STATICPROXY`

| YAML name     | Environment variable  | Required | Default          | Options |
| ------------- | --------------------- | -------- | ---------------- | ------- |
| `staticProxy` | `PIXIVFE_STATICPROXY` | No       | (built-in proxy) | URL     |

The URL of a server that acts as a reverse proxy for s.pximg.net.

### `PIXIVFE_UGOIRAPROXY`

| YAML name     | Environment variable  | Required | Default          | Options |
| ------------- | --------------------- | -------- | ---------------- | ------- |
| `ugoiraProxy` | `PIXIVFE_UGOIRAPROXY` | No       | (built-in proxy) | URL     |

The URL of a server that acts as a reverse proxy for t-hk.ugoira.com.

## Token management

**These options must be nested under a `tokenManager:` block in `config.yml`.**

PixivFE implements exponential backoff for token management to manage rate limiting.

The following environment variables control how PixivFE manages token timeouts when a token encounters repeated failures. The backoff time for a token starts at the base timeout and doubles with each failure, up to the maximum backoff time.

### `PIXIVFE_TOKEN_LOAD_BALANCING`

| YAML name            | Environment variable           | Required | Default       | Options                                        |
| -------------------- | ------------------------------ | -------- | ------------- | ---------------------------------------------- |
| `tokenLoadBalancing` | `PIXIVFE_TOKEN_LOAD_BALANCING` | No       | `round-robin` | `round-robin`, `random`, `least-recently-used` |

Specifies the method for selecting tokens when multiple tokens are provided in `PIXIVFE_TOKEN`.

- `round-robin`: Tokens are used in a circular order.
- `random`: A random token is selected for each request.
- `least-recently-used`: The token that hasn't been used for the longest time is selected.

This option is useful when you have multiple pixiv accounts and want to distribute the load across them.

### `PIXIVFE_TOKEN_MAX_RETRIES`

| YAML name         | Environment variable        | Required | Default | Options |
| ----------------- | --------------------------- | -------- | ------- | ------- |
| `tokenMaxRetries` | `PIXIVFE_TOKEN_MAX_RETRIES` | No       | `5`     | Integer |

Maximum retry attempts before marking a token as unavailable.

### `PIXIVFE_TOKEN_BASE_TIMEOUT`

| YAML name          | Environment variable         | Required | Default  | Options                                                  |
| ------------------ | ---------------------------- | -------- | -------- | -------------------------------------------------------- |
| `tokenBaseTimeout` | `PIXIVFE_TOKEN_BASE_TIMEOUT` | No       | `1000ms` | [`time.Duration`](https://pkg.go.dev/time#ParseDuration) |

Initial backoff duration on token failure.

### `PIXIVFE_TOKEN_MAX_BACKOFF_TIME`

| YAML name             | Environment variable             | Required | Default   | Options                                                  |
| --------------------- | -------------------------------- | -------- | --------- | -------------------------------------------------------- |
| `tokenMaxBackoffTime` | `PIXIVFE_TOKEN_MAX_BACKOFF_TIME` | No       | `32000ms` | [`time.Duration`](https://pkg.go.dev/time#ParseDuration) |

Maximum backoff duration during exponential retry.

## API response caching

**These options must be nested under a `cache:` block in `config.yml`.**

PixivFE implements a caching system for API responses to improve performance. The cache uses a [Least Recently Used (LRU) eviction policy](<https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_Recently_Used_(LRU)>).

Each cache entry is stored with an expiration time. When a cached item is accessed, its expiration time is checked. If the item has expired, it is treated as a cache miss, and a new request is made to the Pixiv API.

To ensure that responses are properly isolated between different users, the cache key for each item is generated based on both the URL of the request and the value of the user's `pixivfe-Token` cookie.

### `PIXIVFE_CACHE_ENABLED`

| YAML name      | Environment variable    | Required | Default | Options |
| -------------- | ----------------------- | -------- | ------- | ------- |
| `cacheEnabled` | `PIXIVFE_CACHE_ENABLED` | No       | `false` | Boolean |

Controls whether the caching system is enabled.

When disabled, all requests will be sent directly to the Pixiv API without caching.

Other caching configuration variables will have no effect if this variable is set to `false`.

### `PIXIVFE_CACHE_SIZE`

| YAML name   | Environment variable | Required | Default | Options |
| ----------- | -------------------- | -------- | ------- | ------- |
| `cacheSize` | `PIXIVFE_CACHE_SIZE` | No       | `100`   | Integer |

Specifies the maximum number of items that can be stored in the LRU cache.

This limits the memory usage of the cache.

When the cache reaches this size, the least recently used items will be evicted to make room for new entries.

### `PIXIVFE_CACHE_TTL`

| YAML name  | Environment variable | Required | Default | Options                                                  |
| ---------- | -------------------- | -------- | ------- | -------------------------------------------------------- |
| `cacheTTL` | `PIXIVFE_CACHE_TTL`  | No       | `60m`   | [`time.Duration`](https://pkg.go.dev/time#ParseDuration) |

Specifies the default Time To Live (TTL) for cached items.

This is the duration for which an item remains valid in the cache before it's considered stale and needs to be fetched again from the pixiv API.

The TTL is applied to most API responses and can safely be set to a high value. Dynamic content such as Discovery and Newest is never cached.

## HTTP caching

**These options must be nested under a `httpCache:` block in `config.yml`.**

These cache control settings affect [HTTP caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching) behavior and are separate from PixivFE's internal API response cache.

### `PIXIVFE_CACHE_CONTROL_MAX_AGE`

| YAML name            | Environment variable            | Required | Default | Options                                                  |
| -------------------- | ------------------------------- | -------- | ------- | -------------------------------------------------------- |
| `cacheControlMaxAge` | `PIXIVFE_CACHE_CONTROL_MAX_AGE` | No       | `30s`   | [`time.Duration`](https://pkg.go.dev/time#ParseDuration) |

Controls the `max-age` directive in the Cache-Control response header for artwork pages.

This determines how long browsers should cache the page before revalidating with the server.

### `PIXIVFE_CACHE_CONTROL_STALE_WHILE_REVALIDATE`

| YAML name                          | Environment variable                           | Required | Default | Options                                                  |
| ---------------------------------- | ---------------------------------------------- | -------- | ------- | -------------------------------------------------------- |
| `cacheControlStaleWhileRevalidate` | `PIXIVFE_CACHE_CONTROL_STALE_WHILE_REVALIDATE` | No       | `60s`   | [`time.Duration`](https://pkg.go.dev/time#ParseDuration) |

Controls the `stale-while-revalidate` directive in the Cache-Control response header for artwork pages.

This allows browsers to show stale content while fetching a fresh version in the background.

## Request parameters

**These options must be nested under a `request:` block in `config.yml`.**

### `PIXIVFE_ACCEPTLANGUAGE`

| YAML name        | Environment variable     | Required | Default          | Options                                                                                                   |
| ---------------- | ------------------------ | -------- | ---------------- | --------------------------------------------------------------------------------------------------------- |
| `acceptLanguage` | `PIXIVFE_ACCEPTLANGUAGE` | No       | `en-US,en;q=0.5` | [Accept-Language value](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Language#syntax) |

The value of the `Accept-Language` header used for requests to pixiv's API.

Change this to modify the response language.

## Response behavior

**These options must be nested under a `response:` block in `config.yml`.**

### `PIXIVFE_EARLY_HINTS_RESPONSES_ENABLED`

| YAML name                    | Environment variable                    | Required | Default | Options |
| ---------------------------- | --------------------------------------- | -------- | ------- | ------- |
| `earlyHintsResponsesEnabled` | `PIXIVFE_EARLY_HINTS_RESPONSES_ENABLED` | No       | `false` | Boolean |

Controls whether PixivFE's internal HTTP server returns [HTTP `103 Early Hints` informational responses](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/103).

Used to return `Link` headers to the client for preloading images while the server is busy preparing the main HTML response.

!!! warning
    Some reverse proxies may not handle informational responses such as HTTP 103 correctly.

### `PIXIVFE_REQUESTLIMIT`

| YAML name      | Environment variable   | Required | Default | Options |
| -------------- | ---------------------- | -------- | ------- | ------- |
| `requestLimit` | `PIXIVFE_REQUESTLIMIT` | No       | -       | Integer |

Set a request limit for the internal HTTP server per 30 seconds.

Set to an integer to enable the built-in rate limiter, e.g., `PIXIVFE_REQUESTLIMIT=15`.

It's recommended to enable rate limiting in the reverse proxy in front of PixivFE rather than using this.

## Features

**These options must be nested under a `feature:` block in `config.yml`.**

### `PIXIVFE_POPULAR_SEARCH_ENABLED`

| YAML name              | Environment variable             | Required | Default | Options |
| ---------------------- | -------------------------------- | -------- | ------- | ------- |
| `popularSearchEnabled` | `PIXIVFE_POPULAR_SEARCH_ENABLED` | No       | `false` | Boolean |

Controls whether searching by popularity for a given tag is enabled.

!!! warning
    This feature requires several API calls for each search.

    API response caching via `PIXIVFE_CACHE_ENABLED=true` is recommended when this is enabled.

## Instance information

**These options must be nested under a `instance:` block in `config.yml`.**

### `PIXIVFE_REPO_URL`

| YAML name | Environment variable | Required | Default                                | Options |
| --------- | -------------------- | -------- | -------------------------------------- | ------- |
| `repoUrl` | `PIXIVFE_REPO_URL`   | No       | `https://codeberg.org/PixivFE/PixivFE` | URL     |

The URL of the PixivFE source code repository.

This is used to provide links to the application source code and specific commit information.

Change this if you're running a fork of PixivFE to link to your own repository instead.

## Network proxy

Used to set the [proxy server](https://en.wikipedia.org/wiki/Proxy_server) that PixivFE will use for all requests.

Not to be confused with the image proxy, which is used to comply with the `Referer` check required by `i.pximg.net`.

Requests use the proxy specified in the configuration option that matches the scheme of the request (`HTTP_PROXY` or `HTTPS_PROXY`).

This selection is based on the scheme of the **request being made**, not on the protocol used by the proxy server itself.

!!! note
    These options are currently only available through environment variables.

### `HTTPS_PROXY`

| YAML name | Environment variable | Required | Default | Options   |
| --------- | -------------------- | -------- | ------- | --------- |
| -         | `HTTPS_PROXY`        | No       | -       | Proxy URL |

Proxy server used for requests made over HTTPS.

### `HTTP_PROXY`

| YAML name | Environment variable | Required | Default | Options   |
| --------- | -------------------- | -------- | ------- | --------- |
| -         | `HTTP_PROXY`         | No       | -       | Proxy URL |

Proxy server used for requests made over plain HTTP.

## Rate limiter

**These options must be nested under a `limiter:` block in `config.yml`.**

PixivFE includes a configurable rate limiter that can help protect your instance from abuse. The limiter uses a network-based approach that groups clients by IP network and applies dynamic rate limits based on client behavior.

### `PIXIVFE_LIMITER_ENABLED`

| YAML name | Environment variable      | Required | Default | Options |
| --------- | ------------------------- | -------- | ------- | ------- |
| `enabled` | `PIXIVFE_LIMITER_ENABLED` | No       | `false` | Boolean |

Controls whether the rate limiter is enabled.

When enabled, the rate limiter middleware will filter HTTP requests based on various criteria and apply rate limits to control traffic. Other limiter options will only take effect if this is set to `true`.

### `PIXIVFE_LIMITER_PASS_IPS`

| YAML name  | Environment variable       | Required | Default | Options                                 |
| ---------- | -------------------------- | -------- | ------- | --------------------------------------- |
| `passList` | `PIXIVFE_LIMITER_PASS_IPS` | No       | -       | Array/comma-separated list of IPs/CIDRs |

A list of IP addresses or CIDR ranges that bypass all rate limiting checks.

Environment variable format:

```sh
PIXIVFE_LIMITER_PASS_IPS=192.168.1.1,10.0.0.0/8,2001:db8::/64
```

Config file format:

```yaml
limiter:
  passList:
    - 192.168.1.1
    - 10.0.0.0/8
    - 2001:db8::/64
```

Requests from these IPs will be allowed without any rate limiting restrictions.

### `PIXIVFE_LIMITER_BLOCK_IPS`

| YAML name   | Environment variable        | Required | Default | Options                                 |
| ----------- | --------------------------- | -------- | ------- | --------------------------------------- |
| `blockList` | `PIXIVFE_LIMITER_BLOCK_IPS` | No       | -       | Array/comma-separated list of IPs/CIDRs |

A list of IP addresses or CIDR ranges that are always blocked.

Environment variable format:

```sh
PIXIVFE_LIMITER_BLOCK_IPS=203.0.113.0/24,2001:db8:1::/48
```

Config file format:

```yaml
limiter:
  blockList:
    - 203.0.113.0/24
    - 2001:db8:1::/48
```

Requests from these IPs will be denied with a [401 Unauthorized](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/401) response.

### `PIXIVFE_LIMITER_FILTER_LOCAL`

| YAML name     | Environment variable           | Required | Default | Options |
| ------------- | ------------------------------ | -------- | ------- | ------- |
| `filterLocal` | `PIXIVFE_LIMITER_FILTER_LOCAL` | No       | `false` | Boolean |

Controls whether link-local addresses (IPv4: `169.254.0.0/16`, IPv6: `fe80::/10`) are subject to rate limiting.

When `true` (enabled), link-local addresses will be rate limited according to the same rules as other addresses. When `false` (disabled, default), link-local addresses will bypass rate limiting checks.

### `PIXIVFE_LIMITER_IPV4_PREFIX`

| YAML name    | Environment variable          | Required | Default | Options        |
| ------------ | ----------------------------- | -------- | ------- | -------------- |
| `ipv4Prefix` | `PIXIVFE_LIMITER_IPV4_PREFIX` | No       | `24`    | Integer (0-32) |

The network prefix length used to group IPv4 addresses for rate limiting purposes.

For example, with the default value of `24`, all addresses within the same `/24` network (e.g., `192.168.1.0/24`) share the same rate limit bucket.

Higher values provide more granular control via smaller groups. Lower values group more addresses together.

### `PIXIVFE_LIMITER_IPV6_PREFIX`

| YAML name    | Environment variable          | Required | Default | Options         |
| ------------ | ----------------------------- | -------- | ------- | --------------- |
| `ipv6Prefix` | `PIXIVFE_LIMITER_IPV6_PREFIX` | No       | `48`    | Integer (0-128) |

The network prefix length used to group IPv6 addresses for rate limiting purposes.

For example, with the default value of `48`, all addresses within the same `/48` network share the same rate limit bucket.

Higher values provide more granular control via smaller groups. Lower values group more addresses together.

### `PIXIVFE_LIMITER_CHECK_HEADERS`

| YAML name      | Environment variable            | Required | Default | Options |
| -------------- | ------------------------------- | -------- | ------- | ------- |
| `checkHeaders` | `PIXIVFE_LIMITER_CHECK_HEADERS` | No       | `true`  | Boolean |

Controls whether specific HTTP request headers are checked for patterns commonly associated with bots.

If enabled, clients with matching headers will be blocked.

### `PIXIVFE_LIMITER_DETECTION_METHOD`

| YAML name         | Environment variable               | Required | Default     | Options                               |
| ----------------- | ---------------------------------- | -------- | ----------- | ------------------------------------- |
| `detectionMethod` | `PIXIVFE_LIMITER_DETECTION_METHOD` | No       | `""` (none) | `""` (none), `linktoken`, `turnstile` |

Specifies the bot detection method to use. This method determines how clients are verified and whether they are marked as "suspicious," which can lead to stricter rate limits.

- `""` (or not set, effectively 'none'): No specific challenge method is used. Clients are always treated as non-suspicious, but clients can still be blocked by headers if `PIXIVFE_LIMITER_CHECK_HEADERS` is enabled.
- `linktoken`: Enables the "link token" bot detection logic. PixivFE embeds a unique token in HTML pages as a CSS resource. Real browsers will fetch this resource and receive a `pixivfe-Ping` cookie that proves they can properly render web pages. Requests without a valid cookie are considered suspicious and limited more heavily. This helps distinguish between actual users and simple bots. Requires `PIXIVFE_LIMITER_PING_HMAC`.
- `turnstile`: Uses [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/) for bot detection. Clients will be presented with a Turnstile challenge. Upon successful verification, a `pixivfe-Ping` cookie is set. Requests without a valid cookie are considered suspicious and limited more heavily. Requires `PIXIVFE_LIMITER_PING_HMAC`, `PIXIVFE_LIMITER_TURNSTILE_SITE_KEY`, and `PIXIVFE_LIMITER_TURNSTILE_SECRET_KEY`.

### `PIXIVFE_LIMITER_PING_HMAC`

| YAML name  | Environment variable        | Required                                                 | Default | Options                 |
| ---------- | --------------------------- | -------------------------------------------------------- | ------- | ----------------------- |
| `pingHMAC` | `PIXIVFE_LIMITER_PING_HMAC` | Yes (if `detectionMethod` is `linktoken` or `turnstile`) | -       | String (32+ characters) |

The secret key used to sign and verify `pixivfe-Ping` cookies. These cookies are used by both the `linktoken` and `turnstile` detection methods to remember successfully verified clients.

This value must be at least 32 characters long and should be kept secret.

!!! warning
    This setting is required and must be set if `detectionMethod` is `linktoken` or `turnstile`.

    Failure to set this will prevent PixivFE from starting when these methods are active.

### `PIXIVFE_LIMITER_TURNSTILE_SITEKEY`

| YAML name          | Environment variable                | Required                                  | Default | Options |
| ------------------ | ----------------------------------- | ----------------------------------------- | ------- | ------- |
| `turnstileSitekey` | `PIXIVFE_LIMITER_TURNSTILE_SITEKEY` | Yes (if `detectionMethod` is `turnstile`) | -       | String  |

The [sitekey](https://developers.cloudflare.com/turnstile/get-started/#get-a-sitekey-and-secret-key) for Cloudflare Turnstile. This is required if `detectionMethod` is set to `turnstile`.

You can obtain this from your Cloudflare dashboard when setting up a Turnstile widget.

### `PIXIVFE_LIMITER_TURNSTILE_SECRET_KEY`

| YAML name            | Environment variable                   | Required                                  | Default | Options |
| -------------------- | -------------------------------------- | ----------------------------------------- | ------- | ------- |
| `turnstileSecretKey` | `PIXIVFE_LIMITER_TURNSTILE_SECRET_KEY` | Yes (if `detectionMethod` is `turnstile`) | -       | String  |

The [secret key](https://developers.cloudflare.com/turnstile/get-started/#get-a-sitekey-and-secret-key) for Cloudflare Turnstile. This is required if `detectionMethod` is set to `turnstile`.

You can obtain this from your Cloudflare dashboard when setting up a Turnstile widget. This key should be kept confidential.

## Logging

**These options must be nested under a `log:` block in `config.yml`.**

Options to configure [uber-go/zap](https://github.com/uber-go/zap), which PixivFE uses for structured logging.

### `PIXIVFE_LOG_LEVEL`

| YAML name  | Environment variable | Required | Default | Options                          |
| ---------- | -------------------- | -------- | ------- | -------------------------------- |
| `logLevel` | `PIXIVFE_LOG_LEVEL`  | No       | `info`  | `debug`, `info`, `warn`, `error` |

Sets the minimum level of log messages to output.

- `debug`: Includes all log messages, including detailed debug information.
- `info`: Includes informational messages, warnings, and errors.
- `warn`: Includes only warning and error messages.
- `error`: Includes only error messages.

### `PIXIVFE_LOG_OUTPUTS`

| YAML name    | Environment variable  | Required | Default  | Options                       |
| ------------ | --------------------- | -------- | -------- | ----------------------------- |
| `logOutputs` | `PIXIVFE_LOG_OUTPUTS` | No       | `stdout` | `stdout`, `stderr`, file path |

Specifies where log messages should be sent.

This can be a single output or multiple outputs separated by commas (e.g. `stdout,/var/log/pixivfe.log`).

- `stdout`: Standard output (console)
- `stderr`: Standard error
- File paths: e.g., `/var/log/pixivfe.log`

### `PIXIVFE_LOG_FORMAT`

| YAML name   | Environment variable | Required | Default   | Options           |
| ----------- | -------------------- | -------- | --------- | ----------------- |
| `logFormat` | `PIXIVFE_LOG_FORMAT` | No       | `console` | `console`, `json` |

Determines the format of log messages.

- `console`: Human-readable format suitable for console output
- `json`: Structured JSON format, useful for log parsing and analysis tools

## Development

**These options must be nested under a `development:` block in `config.yml`.**

### `PIXIVFE_DEV`

| YAML name       | Environment variable | Required | Default | Options |
| --------------- | -------------------- | -------- | ------- | ------- |
| `inDevelopment` | `PIXIVFE_DEV`        | No       | `false` | Boolean |

Set to any value to enable development mode, e.g., `PIXIVFE_DEV=true`.

In development mode:

1. The server will live-reload HTML templates.
2. Responses are saved to `PIXIVFE_RESPONSE_SAVE_LOCATION`.

### `PIXIVFE_RESPONSE_SAVE_LOCATION`

| YAML name              | Environment variable             | Required | Default                  | Options   |
| ---------------------- | -------------------------------- | -------- | ------------------------ | --------- |
| `responseSaveLocation` | `PIXIVFE_RESPONSE_SAVE_LOCATION` | No       | `/tmp/pixivfe/responses` | File path |

Defines where responses from the pixiv API are saved when in development mode.
