package validator

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator
type Validator struct {
	validate *validator.Validate
}

// New creates a new validator instance
func New() *Validator {
	v := validator.New()

	// Register custom validations
	v.RegisterValidation("phoneE164", validatePhoneE164)
	v.RegisterValidation("latitude", validateLatitude)
	v.RegisterValidation("longitude", validateLongitude)
	v.RegisterValidation("uuid", validateUUID)
	v.RegisterValidation("status", validateStatus)

	// Use JSON tag names in error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{validate: v}
}

// ValidateStruct validates a struct
func (v *Validator) ValidateStruct(obj interface{}) error {
	return v.validate.Struct(obj)
}

// ValidationErrors converts validator errors to a readable format
func ValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()
			param := e.Param()

			switch tag {
			case "required":
				errors[field] = fmt.Sprintf("%s is required", field)
			case "email":
				errors[field] = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				errors[field] = fmt.Sprintf("%s must be at least %s characters", field, param)
			case "max":
				errors[field] = fmt.Sprintf("%s must be at most %s characters", field, param)
			case "gte":
				errors[field] = fmt.Sprintf("%s must be greater than or equal to %s", field, param)
			case "lte":
				errors[field] = fmt.Sprintf("%s must be less than or equal to %s", field, param)
			case "gt":
				errors[field] = fmt.Sprintf("%s must be greater than %s", field, param)
			case "lt":
				errors[field] = fmt.Sprintf("%s must be less than %s", field, param)
			case "oneof":
				errors[field] = fmt.Sprintf("%s must be one of: %s", field, param)
			case "phoneE164":
				errors[field] = fmt.Sprintf("%s must be a valid phone number in E.164 format", field)
			case "latitude":
				errors[field] = fmt.Sprintf("%s must be a valid latitude (-90 to 90)", field)
			case "longitude":
				errors[field] = fmt.Sprintf("%s must be a valid longitude (-180 to 180)", field)
			case "uuid":
				errors[field] = fmt.Sprintf("%s must be a valid UUID", field)
			case "status":
				errors[field] = fmt.Sprintf("%s must be a valid status", field)
			default:
				errors[field] = fmt.Sprintf("%s failed validation for tag: %s", field, tag)
			}
		}
	}

	return errors
}

// Middleware creates a validation middleware
func Middleware(v *Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store validator in context
		c.Set("validator", v)
		c.Next()
	}
}

// ValidateJSON validates JSON request body
func ValidateJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
			"details": err.Error(),
		})
		return false
	}

	// Get validator from context
	validatorInterface, exists := c.Get("validator")
	if !exists {
		// Fallback to default validator
		validatorInterface = New()
	}

	v, ok := validatorInterface.(*Validator)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Validator configuration error",
		})
		return false
	}

	if err := v.ValidateStruct(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Validation failed",
			"details": ValidationErrors(err),
		})
		return false
	}

	return true
}

// Custom validation functions

func validatePhoneE164(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// Simple E.164 validation: +[country code][number]
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}
	return strings.HasPrefix(phone, "+")
}

func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90 && lat <= 90
}

func validateLongitude(fl validator.FieldLevel) bool {
	lng := fl.Field().Float()
	return lng >= -180 && lng <= 180
}

func validateUUID(fl validator.FieldLevel) bool {
	uuid := fl.Field().String()
	// Simple UUID validation (format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	if len(uuid) != 36 {
		return false
	}
	// Check for hyphens in correct positions
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	return true
}

func validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	// This is a placeholder - should be customized per service
	validStatuses := []string{
		"PENDING", "CONFIRMED", "PROCESSING", "COMPLETED",
		"CANCELLED", "FAILED", "SHIPPED", "DELIVERED",
		"AVAILABLE", "ASSIGNED", "OFF_DUTY", "IN_PROGRESS",
		"PLANNED", "SENT",
	}

	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// Common validation structs

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int `form:"page" json:"page" validate:"omitempty,gte=1"`
	PageSize int `form:"page_size" json:"page_size" validate:"omitempty,gte=1,lte=100"`
}

// GetPagination returns pagination with defaults
func (p *PaginationRequest) GetPagination() (page, pageSize int) {
	page = p.Page
	if page < 1 {
		page = 1
	}

	pageSize = p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return page, pageSize
}

// IDRequest represents a request with an ID
type IDRequest struct {
	ID string `uri:"id" json:"id" validate:"required,uuid"`
}

// Location represents geographical coordinates
type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

// Email represents an email field
type Email struct {
	Email string `json:"email" validate:"required,email"`
}

// Phone represents a phone field
type Phone struct {
	Phone string `json:"phone" validate:"required,phoneE164"`
}
