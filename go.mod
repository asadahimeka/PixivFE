module codeberg.org/pixivfe/pixivfe

go 1.24

// NOTE: remember to update Dockerfiles and CI definitions when bumping this
toolchain go1.24.3

// for go-jnode
replace encoding/json => github.com/goccy/go-json v0.10.5

require (
	github.com/CloudyKit/jet/v6 v6.3.1
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/a-h/templ v0.3.865
	github.com/goccy/go-json v0.10.5
	github.com/gorilla/mux v1.8.1
	github.com/oklog/ulid/v2 v2.1.1
	github.com/sethvargo/go-envconfig v1.3.0
	github.com/soluble-ai/go-jnode v0.1.11
	github.com/tdewolff/minify/v2 v2.23.8
	github.com/tidwall/gjson v1.18.0
	github.com/timandy/routine v1.1.5
	github.com/zeebo/xxh3 v1.0.2
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.40.0
	golang.org/x/sync v0.14.0
	golang.org/x/time v0.11.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/tdewolff/parse/v2 v2.8.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/zeebo/assert v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/tools v0.32.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

tool github.com/a-h/templ/cmd/templ
