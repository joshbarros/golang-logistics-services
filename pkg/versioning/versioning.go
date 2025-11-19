package versioning

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Strategy defines how API version is extracted from request
type Strategy string

const (
	// StrategyPath extracts version from URL path (/api/v1/orders)
	StrategyPath Strategy = "path"
	// StrategyHeader extracts version from custom header (X-API-Version: 1)
	StrategyHeader Strategy = "header"
	// StrategyQueryParam extracts version from query param (?version=1)
	StrategyQueryParam Strategy = "query"
	// StrategyAcceptHeader extracts version from Accept header (application/vnd.api.v1+json)
	StrategyAcceptHeader Strategy = "accept"
)

// Config holds versioning configuration
type Config struct {
	// Strategy defines how to extract version
	Strategy Strategy
	// DefaultVersion is used when no version is specified
	DefaultVersion int
	// MinVersion is the minimum supported version
	MinVersion int
	// MaxVersion is the maximum supported version
	MaxVersion int
	// HeaderName is the header name for header strategy (default: "X-API-Version")
	HeaderName string
	// QueryParamName is the query param name for query strategy (default: "version")
	QueryParamName string
	// DeprecatedVersions lists versions that are deprecated
	DeprecatedVersions []int
	// OnDeprecated is called when a deprecated version is used
	OnDeprecated func(c *gin.Context, version int)
}

// DefaultConfig returns default versioning configuration
func DefaultConfig() Config {
	return Config{
		Strategy:       StrategyPath,
		DefaultVersion: 1,
		MinVersion:     1,
		MaxVersion:     1,
		HeaderName:     "X-API-Version",
		QueryParamName: "version",
	}
}

// Middleware returns a Gin middleware that extracts and validates API version
func Middleware(config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		version, err := extractVersion(c, config)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid API version",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate version range
		if version < config.MinVersion || version > config.MaxVersion {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unsupported API version",
				"message": fmt.Sprintf("Supported versions: %d-%d",
					config.MinVersion, config.MaxVersion),
				"requested_version": version,
			})
			c.Abort()
			return
		}

		// Check if deprecated
		if isDeprecated(version, config.DeprecatedVersions) {
			// Add deprecation warning header
			c.Header("X-API-Deprecated", "true")
			c.Header("X-API-Deprecation-Info",
				fmt.Sprintf("Version %d is deprecated. Please upgrade to version %d",
					version, config.MaxVersion))

			// Call deprecated callback
			if config.OnDeprecated != nil {
				config.OnDeprecated(c, version)
			}
		}

		// Store version in context
		c.Set("api_version", version)

		// Add version header to response
		c.Header("X-API-Version", strconv.Itoa(version))

		c.Next()
	}
}

// extractVersion extracts version from request based on strategy
func extractVersion(c *gin.Context, config Config) (int, error) {
	switch config.Strategy {
	case StrategyPath:
		return extractFromPath(c)
	case StrategyHeader:
		return extractFromHeader(c, config.HeaderName, config.DefaultVersion)
	case StrategyQueryParam:
		return extractFromQuery(c, config.QueryParamName, config.DefaultVersion)
	case StrategyAcceptHeader:
		return extractFromAcceptHeader(c, config.DefaultVersion)
	default:
		return config.DefaultVersion, nil
	}
}

// extractFromPath extracts version from URL path
// Supports formats: /api/v1/... or /v1/api/...
func extractFromPath(c *gin.Context) (int, error) {
	path := c.Request.URL.Path
	re := regexp.MustCompile(`/v(\d+)/`)
	matches := re.FindStringSubmatch(path)

	if len(matches) < 2 {
		return 0, fmt.Errorf("version not found in path")
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid version in path: %s", matches[1])
	}

	return version, nil
}

// extractFromHeader extracts version from custom header
func extractFromHeader(c *gin.Context, headerName string, defaultVersion int) (int, error) {
	versionStr := c.GetHeader(headerName)
	if versionStr == "" {
		return defaultVersion, nil
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return 0, fmt.Errorf("invalid version in header: %s", versionStr)
	}

	return version, nil
}

// extractFromQuery extracts version from query parameter
func extractFromQuery(c *gin.Context, paramName string, defaultVersion int) (int, error) {
	versionStr := c.Query(paramName)
	if versionStr == "" {
		return defaultVersion, nil
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return 0, fmt.Errorf("invalid version in query param: %s", versionStr)
	}

	return version, nil
}

// extractFromAcceptHeader extracts version from Accept header
// Supports format: application/vnd.api.v1+json
func extractFromAcceptHeader(c *gin.Context, defaultVersion int) (int, error) {
	accept := c.GetHeader("Accept")
	if accept == "" {
		return defaultVersion, nil
	}

	re := regexp.MustCompile(`application/vnd\..*\.v(\d+)\+json`)
	matches := re.FindStringSubmatch(accept)

	if len(matches) < 2 {
		return defaultVersion, nil
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return defaultVersion, nil
	}

	return version, nil
}

// isDeprecated checks if version is deprecated
func isDeprecated(version int, deprecated []int) bool {
	for _, v := range deprecated {
		if v == version {
			return true
		}
	}
	return false
}

// GetVersion returns the API version from context
func GetVersion(c *gin.Context) int {
	version, exists := c.Get("api_version")
	if !exists {
		return 1 // default
	}
	return version.(int)
}

// VersionedRouter manages multiple API versions
type VersionedRouter struct {
	engine  *gin.Engine
	config  Config
	routers map[int]*gin.RouterGroup
}

// NewVersionedRouter creates a new versioned router
func NewVersionedRouter(engine *gin.Engine, config Config) *VersionedRouter {
	return &VersionedRouter{
		engine:  engine,
		config:  config,
		routers: make(map[int]*gin.RouterGroup),
	}
}

// Version returns a router group for a specific version
func (vr *VersionedRouter) Version(version int) *gin.RouterGroup {
	if router, exists := vr.routers[version]; exists {
		return router
	}

	// Create new version router
	var group *gin.RouterGroup
	switch vr.config.Strategy {
	case StrategyPath:
		group = vr.engine.Group(fmt.Sprintf("/api/v%d", version))
	default:
		group = vr.engine.Group("/api")
	}

	// Add versioning middleware
	group.Use(Middleware(vr.config))

	vr.routers[version] = group
	return group
}

// RegisterHandler registers a handler for a specific version
func (vr *VersionedRouter) RegisterHandler(version int, method, path string, handlers ...gin.HandlerFunc) {
	router := vr.Version(version)
	router.Handle(method, path, handlers...)
}

// RegisterMultiVersion registers a handler for multiple versions
func (vr *VersionedRouter) RegisterMultiVersion(versions []int, method, path string, handlers ...gin.HandlerFunc) {
	for _, version := range versions {
		vr.RegisterHandler(version, method, path, handlers...)
	}
}

// Example usage:
//
// func main() {
//     router := gin.New()
//
//     // Path-based versioning
//     versionedRouter := versioning.NewVersionedRouter(router, versioning.Config{
//         Strategy:           versioning.StrategyPath,
//         DefaultVersion:     2,
//         MinVersion:         1,
//         MaxVersion:         2,
//         DeprecatedVersions: []int{1},
//         OnDeprecated: func(c *gin.Context, version int) {
//             log.Printf("Deprecated API version %d used by %s", version, c.ClientIP())
//         },
//     })
//
//     // V1 endpoints (deprecated)
//     v1 := versionedRouter.Version(1)
//     v1.GET("/orders", getOrdersV1)
//     v1.POST("/orders", createOrderV1)
//
//     // V2 endpoints (current)
//     v2 := versionedRouter.Version(2)
//     v2.GET("/orders", getOrdersV2)
//     v2.POST("/orders", createOrderV2)
//
//     // Or register for multiple versions
//     versionedRouter.RegisterMultiVersion(
//         []int{1, 2},
//         "GET",
//         "/health",
//         healthCheckHandler,
//     )
//
//     router.Run(":8080")
// }
//
// // In handlers, you can check version
// func getOrdersV2(c *gin.Context) {
//     version := versioning.GetVersion(c)
//     if version >= 2 {
//         // Use new response format
//     }
// }
//
// // Header-based versioning
// router := gin.New()
// router.Use(versioning.Middleware(versioning.Config{
//     Strategy:       versioning.StrategyHeader,
//     HeaderName:     "X-API-Version",
//     DefaultVersion: 2,
//     MinVersion:     1,
//     MaxVersion:     2,
// }))
//
// // Accept header versioning
// router.Use(versioning.Middleware(versioning.Config{
//     Strategy:       versioning.StrategyAcceptHeader,
//     DefaultVersion: 1,
//     MinVersion:     1,
//     MaxVersion:     2,
// }))
// // Client sends: Accept: application/vnd.api.v1+json
