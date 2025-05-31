// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // Import the timezone database for when the system timezone database is not available

	// TODO: figure out how to properly implement urlx
	// the implementation in 5f8b659b49 causes config.go to segfault due to a nil pointer dereference when the PIXIVFE_IMAGEPROXY env var is not set
	// "github.com/goware/urlx"
	"codeberg.org/pixivfe/pixivfe/server/tokenmanager"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

var (
	GlobalConfig   ServerConfig // GlobalConfig is a global variable that exposes serverConfig.
	revision       string       // revision stores the current version's revision information.
)

// Placeholder/fallback values.
const (
	unknownRevision string = "unknown"
	revisionFormat  string = "date-hash[+dirty]"
)

// Limiter.DetectionMethod values.
const (
	NoneDetectionMethod      string = ""
	LinkTokenDetectionMethod string = "linktoken"
	TurnstileDetectionMethod string = "turnstile"
)

// Configration defaults.
const (
	version                                 string        = "v3.0.1"
	defaultHost                             string        = "localhost"
	defaultPort                             string        = "8282"
	defaultTimeZone                         string        = "Etc/UTC"
	defaultImageProxyStaging                string        = BuiltInImageProxyPath
	defaultStaticProxyStaging               string        = BuiltInStaticProxyPath
	defaultUgoiraProxyStaging               string        = BuiltInUgoiraProxyPath
	defaultTokenLoadBalancing               string        = "round-robin"
	defaultTokenMaxRetries                  int           = 5
	defaultTokenBaseTimeout                 time.Duration = 1000 * time.Millisecond
	defaultTokenMaxBackoffTime              time.Duration = 32000 * time.Millisecond
	defaultCacheEnabled                     bool          = false
	defaultCacheSize                        int           = 100
	defaultCacheTTL                         time.Duration = 60 * time.Minute
	defaultCacheControlMaxAge               time.Duration = 30 * time.Second
	defaultCacheControlStaleWhileRevalidate time.Duration = 60 * time.Second
	defaultAcceptLanguage                   string        = "en-US,en;q=0.5"
	defaultEarlyHintsResponsesEnabled       bool          = false
	defaultPopularSearchEnabled             bool          = false
	defaultRepoURL                          string        = "https://codeberg.org/PixivFE/PixivFE"
	defaultResponseSaveLocation             string        = "/tmp/pixivfe/responses"
	defaultLogLevel                         string        = "info"
	defaultLogFormat                        string        = "console"
	defaultLimiterEnabled                   bool          = false
	DefaultLimiterFilterLocal               bool          = false
	DefaultLimiterIPv4Prefix                int           = 24 // /24 network
	DefaultLimiterIPv6Prefix                int           = 48 // /48 network
	defaultLimiterCheckHeaders              bool          = true
	defaultLimiterDetectionMethod           string        = ""
)

var defaultLogOutputs = []string{"stdout"}

type ServerConfig struct {
	Basic struct {
		Host         string `env:"PIXIVFE_HOST,overwrite" yaml:"host"`
		Port         string `env:"PIXIVFE_PORT,overwrite" yaml:"port"`
		UnixSocket   string `env:"PIXIVFE_UNIXSOCKET" yaml:"unixSocket"`
		TimeZone     string `env:"PIXIVFE_TZ,overwrite" yaml:"timeZone"`
		TimeLocation *time.Location
		Token        []string `env:"PIXIVFE_TOKEN" yaml:"token"`
	}

	ContentProxies struct {
		RawImage  string  `env:"PIXIVFE_IMAGEPROXY,overwrite" yaml:"imageProxy"`
		Image     url.URL // For i.pximg.net
		RawStatic string  `env:"PIXIVFE_STATICPROXY,overwrite" yaml:"staticProxy"`
		Static    url.URL // For s.pximg.net
		RawUgoira string  `env:"PIXIVFE_UGOIRAPROXY,overwrite" yaml:"ugoiraProxy"`
		Ugoira    url.URL // For ugoira.com
	}

	TokenManager struct {
		TokenManager   *tokenmanager.TokenManager
		LoadBalancing  string        `env:"PIXIVFE_TOKEN_LOAD_BALANCING,overwrite" yaml:"tokenLoadBalancing"`
		MaxRetries     int           `env:"PIXIVFE_TOKEN_MAX_RETRIES,overwrite" yaml:"tokenMaxRetries"`
		BaseTimeout    time.Duration `env:"PIXIVFE_TOKEN_BASE_TIMEOUT,overwrite" yaml:"tokenBaseTimeout"`
		MaxBackoffTime time.Duration `env:"PIXIVFE_TOKEN_MAX_BACKOFF_TIME,overwrite" yaml:"tokenMaxBackoffTime"`
	}

	Cache struct {
		Enabled bool          `env:"PIXIVFE_CACHE_ENABLED,overwrite" yaml:"cacheEnabled"`
		Size    int           `env:"PIXIVFE_CACHE_SIZE,overwrite" yaml:"cacheSize"`
		TTL     time.Duration `env:"PIXIVFE_CACHE_TTL,overwrite" yaml:"cacheTTL"`
	}

	HTTPCache struct {
		MaxAge               time.Duration `env:"PIXIVFE_CACHE_CONTROL_MAX_AGE,overwrite" yaml:"cacheControlMaxAge"`
		StaleWhileRevalidate time.Duration `env:"PIXIVFE_CACHE_CONTROL_STALE_WHILE_REVALIDATE,overwrite" yaml:"cacheControlStaleWhileRevalidate"`
	}

	Request struct {
		AcceptLanguage string `env:"PIXIVFE_ACCEPTLANGUAGE,overwrite" yaml:"acceptLanguage"`
	}

	Response struct {
		EarlyHintsResponsesEnabled bool   `env:"PIXIVFE_EARLY_HINTS_RESPONSES_ENABLED,overwrite" yaml:"earlyHintsResponsesEnabled"`
		RequestLimit               uint64 `env:"PIXIVFE_REQUESTLIMIT" yaml:"requestLimit"`
	}

	Feature struct {
		PopularSearchEnabled bool `env:"PIXIVFE_POPULAR_SEARCH_ENABLED,overwrite" yaml:"popularSearchEnabled"`
	}

	Instance struct {
		Version      string
		Revision     string
		RevisionDate string
		RevisionHash string
		IsDirty      bool
		StartingTime string
		RepoURL      string `env:"PIXIVFE_REPO_URL,overwrite" yaml:"repoUrl"`
	}

	Development struct {
		InDevelopment        bool   `env:"PIXIVFE_DEV" yaml:"inDevelopment"`
		ResponseSaveLocation string `env:"PIXIVFE_RESPONSE_SAVE_LOCATION,overwrite" yaml:"responseSaveLocation"`
	}

	Log struct {
		Level   string   `env:"PIXIVFE_LOG_LEVEL,overwrite" yaml:"logLevel"`
		Outputs []string `env:"PIXIVFE_LOG_OUTPUTS,overwrite" yaml:"logOutputs"`
		Format  string   `env:"PIXIVFE_LOG_FORMAT,overwrite" yaml:"logFormat"`
	}

	Limiter struct {
		Enabled            bool     `env:"PIXIVFE_LIMITER_ENABLED,overwrite" yaml:"enabled"`
		PassIPs            []string `env:"PIXIVFE_LIMITER_PASS_IPS,overwrite" yaml:"passList"`
		BlockIPs           []string `env:"PIXIVFE_LIMITER_BLOCK_IPS,overwrite" yaml:"blockList"`
		FilterLocal        bool     `env:"PIXIVFE_LIMITER_FILTER_LOCAL,overwrite" yaml:"filterLocal"`
		IPv4Prefix         int      `env:"PIXIVFE_LIMITER_IPV4_PREFIX,overwrite" yaml:"ipv4Prefix"`
		IPv6Prefix         int      `env:"PIXIVFE_LIMITER_IPV6_PREFIX,overwrite" yaml:"ipv6Prefix"`
		CheckHeaders       bool     `env:"PIXIVFE_LIMITER_CHECK_HEADERS,overwrite" yaml:"checkHeaders"`
		DetectionMethod    string   `env:"PIXIVFE_LIMITER_DETECTION_METHOD,overwrite" yaml:"detectionMethod"`
		PingHMAC           string   `env:"PIXIVFE_LIMITER_PING_HMAC" yaml:"pingHMAC"`
		TurnstileSitekey   string   `env:"PIXIVFE_LIMITER_TURNSTILE_SITEKEY" yaml:"turnstileSitekey"`
		TurnstileSecretKey string   `env:"PIXIVFE_LIMITER_TURNSTILE_SECRET_KEY" yaml:"turnstileSecretKey"`
	}
}

func (cfg *ServerConfig) GetToken() string {
	token := cfg.TokenManager.TokenManager.GetToken()
	if token == nil {
		log.Println("[WARNING] All tokens are timed out. Using the first available token.")

		return cfg.Basic.Token[0]
	}

	return token.Value
}

func (cfg *ServerConfig) GetP_AB() string {
	token := cfg.TokenManager.TokenManager.GetToken()
	if token == nil {
		log.Println("[WARNING] All tokens are timed out. Using the first available token.")

		return cfg.Basic.Token[0]
	}

	return token.P_AB
}

func (cfg *ServerConfig) LoadConfig() error {
	parseCommandLineArgs()

	// Override config file path with environment variable if set
	if envVar := os.Getenv("PIXIVFE_CONFIGFILE"); envVar != "" {
		configFilePath = envVar
	}

	cfg.setInstanceInfo()

	cfg.loadFromDefaults()

	if err := cfg.loadFromYAML(); err != nil {
		return fmt.Errorf("error loading YAML config: %w", err)
	}

	if err := cfg.loadFromEnv(); err != nil {
		return fmt.Errorf("error loading environment variables: %w", err)
	}

	if err := cfg.validateAndSet(); err != nil {
		return fmt.Errorf("configuration validation and setting failed: %w", err)
	}

	cfg.initComponents()

	cfg.printConfiguration()

	return nil
}

func (cfg *ServerConfig) setInstanceInfo() {
	cfg.Instance.Version = version
	cfg.Instance.Revision = revision
	cfg.Instance.RevisionDate, cfg.Instance.RevisionHash, cfg.Instance.IsDirty = parseRevision(revision)
	cfg.Instance.StartingTime = time.Now().UTC().Format("2006-01-02 15:04")

	log.Printf("PixivFE %s, revision %s\n", cfg.Instance.Version, cfg.Instance.Revision)

	if revision == "" {
		log.Printf("[WARNING] REVISION is not set. Continuing with unknown revision information.\n")
	} else if cfg.Instance.RevisionDate == unknownRevision {
		log.Printf("[WARNING] REVISION format is invalid: %s. Expected format '%s'. Continuing with full revision as hash.\n", revision, revisionFormat)
	}
}

func (cfg *ServerConfig) loadFromDefaults() {
	cfg.Basic.Host = defaultHost
	cfg.Basic.Port = defaultPort
	cfg.Basic.TimeZone = defaultTimeZone
	cfg.Instance.RepoURL = defaultRepoURL
	cfg.Request.AcceptLanguage = defaultAcceptLanguage
	cfg.ContentProxies.RawImage = defaultImageProxyStaging
	cfg.ContentProxies.RawStatic = defaultStaticProxyStaging
	cfg.ContentProxies.RawUgoira = defaultUgoiraProxyStaging
	cfg.TokenManager.LoadBalancing = defaultTokenLoadBalancing
	cfg.TokenManager.MaxRetries = defaultTokenMaxRetries
	cfg.TokenManager.BaseTimeout = defaultTokenBaseTimeout
	cfg.TokenManager.MaxBackoffTime = defaultTokenMaxBackoffTime
	cfg.Cache.Enabled = defaultCacheEnabled
	cfg.Cache.Size = defaultCacheSize
	cfg.Cache.TTL = defaultCacheTTL
	cfg.HTTPCache.MaxAge = defaultCacheControlMaxAge
	cfg.HTTPCache.StaleWhileRevalidate = defaultCacheControlStaleWhileRevalidate
	cfg.Response.EarlyHintsResponsesEnabled = defaultEarlyHintsResponsesEnabled
	cfg.Feature.PopularSearchEnabled = defaultPopularSearchEnabled
	cfg.Development.ResponseSaveLocation = defaultResponseSaveLocation
	cfg.Log.Level = defaultLogLevel
	cfg.Log.Outputs = defaultLogOutputs
	cfg.Log.Format = defaultLogFormat
	cfg.Limiter.Enabled = defaultLimiterEnabled
	cfg.Limiter.CheckHeaders = defaultLimiterCheckHeaders
	cfg.Limiter.DetectionMethod = defaultLimiterDetectionMethod
	cfg.Limiter.FilterLocal = DefaultLimiterFilterLocal
	cfg.Limiter.IPv4Prefix = DefaultLimiterIPv4Prefix
	cfg.Limiter.IPv6Prefix = DefaultLimiterIPv6Prefix
}

func (cfg *ServerConfig) loadFromYAML() error {
	if configFilePath == "" {
		return nil
	}

	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		log.Printf("No configuration file found at %s, skipping YAML config\n", configFilePath)

		return nil
	}

	ymlConf, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %w", configFilePath, err)
	}

	if err := yaml.Unmarshal(ymlConf, cfg); err != nil {
		return fmt.Errorf("failed to parse YAML from %s: %w", configFilePath, err)
	}

	log.Printf("Successfully loaded configuration from %s\n", configFilePath)

	return nil
}

func (cfg *ServerConfig) loadFromEnv() error {
	return envconfig.Process(context.Background(), cfg)
}

func (cfg *ServerConfig) validateAndSet() error {
	// Handle listener configuration
	if cfg.Basic.UnixSocket != "" {
		if cfg.Basic.Host != "" || cfg.Basic.Port != "" {
			return fmt.Errorf("unix socket configured - cannot specify Host (%s) and Port (%s) simultaneously",
				cfg.Basic.Host, cfg.Basic.Port)
		}
	} else {
		// Set TCP defaults
		if cfg.Basic.Host == "" {
			cfg.Basic.Host = defaultHost
			log.Printf("[INFO] Binding to default host: %s\n", cfg.Basic.Host)
		}

		if cfg.Basic.Port == "" {
			cfg.Basic.Port = defaultPort
			log.Printf("[INFO] Using default port: %s\n", cfg.Basic.Port)
		}
	}

	// Validate TimeZone
	location, err := time.LoadLocation(cfg.Basic.TimeZone)
	if err != nil {
		return fmt.Errorf("invalid time zone '%s': %w", cfg.Basic.TimeZone, err)
	}

	cfg.Basic.TimeLocation = location

	// Check tokens
	if len(cfg.Basic.Token) == 0 {
		return errors.New("no token supplied. Please supply at least one token")
	}

	// Validate image proxy
	if err := validateProxy(&cfg.ContentProxies.RawImage, BuiltInImageProxyPath, "image"); err != nil {
		return err
	}

	if cfg.ContentProxies.RawImage == BuiltInImageProxyPath {
		cfg.ContentProxies.Image = url.URL{Path: BuiltInImageProxyPath}
	} else {
		parsedURL, _ := url.Parse(cfg.ContentProxies.RawImage)
		cfg.ContentProxies.Image = *parsedURL
	}

	// Validate static proxy
	if err := validateProxy(&cfg.ContentProxies.RawStatic, BuiltInStaticProxyPath, "static"); err != nil {
		return err
	}

	if cfg.ContentProxies.RawStatic == BuiltInStaticProxyPath {
		cfg.ContentProxies.Static = url.URL{Path: BuiltInStaticProxyPath}
	} else {
		parsedURL, _ := url.Parse(cfg.ContentProxies.RawStatic)
		cfg.ContentProxies.Static = *parsedURL
	}

	// Validate ugoira proxy
	if err := validateProxy(&cfg.ContentProxies.RawUgoira, BuiltInUgoiraProxyPath, "ugoira"); err != nil {
		return err
	}

	if cfg.ContentProxies.RawUgoira == BuiltInUgoiraProxyPath {
		cfg.ContentProxies.Ugoira = url.URL{Path: BuiltInUgoiraProxyPath}
	} else {
		parsedURL, _ := url.Parse(cfg.ContentProxies.RawUgoira)
		cfg.ContentProxies.Ugoira = *parsedURL
	}

	// Validate RepoURL
	repoURL, err := utils.ValidateURL(cfg.Instance.RepoURL, "Repo")
	if err != nil {
		return fmt.Errorf("invalid repo URL: %w", err)
	}

	cfg.Instance.RepoURL = repoURL.String()

	// Validate TokenLoadBalancing
	switch cfg.TokenManager.LoadBalancing {
	case "round-robin", "random", "least-recently-used":
		// valid
	default:
		return fmt.Errorf("invalid TokenLoadBalancing value: %s (must be 'round-robin', 'random', or 'least-recently-used')",
			cfg.TokenManager.LoadBalancing)
	}

	// Skip validating Limiter configuration if it's not enabled
	if !cfg.Limiter.Enabled {
		return nil
	}

	// Validate DetectionMethod
	switch cfg.Limiter.DetectionMethod {
	case NoneDetectionMethod, LinkTokenDetectionMethod, TurnstileDetectionMethod:
		// valid
	default:
		return fmt.Errorf(
			"invalid Limiter.DetectionMethod: '%s'. Must be one of %q, %q, or %q",
			cfg.Limiter.DetectionMethod,
			NoneDetectionMethod,
			LinkTokenDetectionMethod,
			TurnstileDetectionMethod,
		)
	}

	// PingHMAC is required if a detection method that uses it is enabled
	if cfg.Limiter.DetectionMethod == LinkTokenDetectionMethod || cfg.Limiter.DetectionMethod == TurnstileDetectionMethod {
		if cfg.Limiter.PingHMAC == "" {
			return errors.New(
				"Limiter.PingHMAC is required when Limiter.DetectionMethod is " +
					LinkTokenDetectionMethod + " or " + TurnstileDetectionMethod,
			)
		}

		if len(cfg.Limiter.PingHMAC) < 32 {
			return errors.New("Limiter.PingHMAC must be at least 32 characters long")
		}
	}

	// Turnstile specific configuration
	if cfg.Limiter.DetectionMethod == TurnstileDetectionMethod {
		if cfg.Limiter.TurnstileSitekey == "" {
			return errors.New(
				"Limiter.TurnstileSiteKey is required when Limiter.DetectionMethod is " +
					TurnstileDetectionMethod,
			)
		}

		if cfg.Limiter.TurnstileSecretKey == "" {
			return errors.New(
				"Limiter.TurnstileSecretKey is required when Limiter.DetectionMethod is " +
					TurnstileDetectionMethod,
			)
		}
	}

	if cfg.Limiter.IPv4Prefix < 0 || cfg.Limiter.IPv4Prefix > 32 {
		return fmt.Errorf("IPv4 prefix must be between 0 and 32, got %d", cfg.Limiter.IPv4Prefix)
	}

	if cfg.Limiter.IPv6Prefix < 0 || cfg.Limiter.IPv6Prefix > 128 {
		return fmt.Errorf("IPv6 prefix must be between 0 and 128, got %d", cfg.Limiter.IPv6Prefix)
	}

	return nil
}

func (cfg *ServerConfig) initComponents() {
	cfg.TokenManager.TokenManager = tokenmanager.NewTokenManager(cfg.Basic.Token, cfg.TokenManager.MaxRetries, cfg.TokenManager.BaseTimeout, cfg.TokenManager.MaxBackoffTime, cfg.TokenManager.LoadBalancing)
}

// parseRevision extracts RevisionDate, RevisionHash, and IsDirty status from the Revision string.
func parseRevision(revision string) (string, string, bool) {
	// Check if Revision is empty
	if revision == "" {
		return unknownRevision, unknownRevision, false
	}

	// Check if Revision is marked as dirty (has uncommitted changes)
	isDirty := strings.HasSuffix(revision, "+dirty")
	if isDirty {
		revision = strings.TrimSuffix(revision, "+dirty")
	}

	// Split Revision into two parts, RevisionDate and RevisionHash
	parts := strings.Split(revision, "-")
	if len(parts) == 2 {
		return parts[0], parts[1], isDirty
	}

	// Return unknown date, full string as hash if format doesn't match date-hash
	return unknownRevision, revision, isDirty
}

// validateProxy validates ContentProxies.
func validateProxy(rawURL *string, defaultPath string, proxyType string) error {
	if *rawURL == defaultPath {
		return nil
	}

	_, err := utils.ValidateURL(*rawURL, proxyType+" proxy server")
	if err != nil {
		return fmt.Errorf("invalid %s proxy URL: %w", proxyType, err)
	}

	return nil
}
