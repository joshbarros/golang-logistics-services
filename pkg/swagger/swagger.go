package swagger

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Config holds Swagger configuration
type Config struct {
	// Title of the API
	Title string
	// Description of the API
	Description string
	// Version of the API
	Version string
	// Host address (e.g., "localhost:8080" or "api.example.com")
	Host string
	// BasePath for the API (e.g., "/api/v1")
	BasePath string
	// Schemes (http, https)
	Schemes []string
	// Contact information
	ContactName  string
	ContactEmail string
	ContactURL   string
	// License information
	LicenseName string
	LicenseURL  string
}

// SetupSwagger configures Swagger/OpenAPI documentation endpoint
// It serves the Swagger UI at /swagger/*any
func SetupSwagger(router *gin.Engine, config Config) {
	// Swagger UI endpoint
	url := ginSwagger.URL("/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}

// Example usage in main.go:
//
// import (
//     "github.com/joshbarros/golang-logistics-services/pkg/swagger"
//     _ "github.com/joshbarros/golang-logistics-services/services/order-service/docs" // Import generated docs
// )
//
// // @title Order Service API
// // @version 2.0.0
// // @description Enterprise-grade order management microservice
// // @termsOfService http://swagger.io/terms/
//
// // @contact.name API Support
// // @contact.url http://www.example.com/support
// // @contact.email support@example.com
//
// // @license.name Apache 2.0
// // @license.url http://www.apache.org/licenses/LICENSE-2.0.html
//
// // @host localhost:8080
// // @BasePath /api/v1
//
// // @securityDefinitions.apikey BearerAuth
// // @in header
// // @name Authorization
// // @description Type "Bearer" followed by a space and JWT token.
//
// func main() {
//     router := gin.New()
//     swagger.SetupSwagger(router, swagger.Config{
//         Title:       "Order Service API",
//         Description: "Enterprise-grade order management",
//         Version:     "2.0.0",
//         Host:        "localhost:8080",
//         BasePath:    "/api/v1",
//     })
// }

// Example handler documentation:
//
// // CreateOrder godoc
// // @Summary Create a new order
// // @Description Create a new order with the provided details
// // @Tags orders
// // @Accept json
// // @Produce json
// // @Param order body CreateOrderRequest true "Order details"
// // @Success 201 {object} CreateOrderResponse
// // @Failure 400 {object} ErrorResponse
// // @Failure 401 {object} ErrorResponse "Unauthorized"
// // @Failure 403 {object} ErrorResponse "Forbidden - insufficient permissions"
// // @Failure 500 {object} ErrorResponse
// // @Security BearerAuth
// // @Router /orders [post]
// func (h *HTTPHandler) CreateOrder(c *gin.Context) {
//     // implementation
// }

// Common response types for documentation

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid input"`
	Message string `json:"message,omitempty" example:"The request body is invalid"`
	Code    string `json:"code,omitempty" example:"INVALID_REQUEST"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string      `json:"message" example:"Operation completed successfully"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	TotalCount int64       `json:"total_count" example:"100"`
	Page       int         `json:"page" example:"1"`
	PageSize   int         `json:"page_size" example:"10"`
	TotalPages int         `json:"total_pages" example:"10"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status" example:"healthy"`
	Service   string            `json:"service" example:"order-service"`
	Version   string            `json:"version" example:"2.0.0"`
	Timestamp string            `json:"timestamp" example:"2025-01-19T10:00:00Z"`
	Checks    map[string]string `json:"checks,omitempty"`
}
