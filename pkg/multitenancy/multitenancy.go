package multitenancy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrTenantNotFound is returned when tenant is not found
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrNoTenant is returned when no tenant in context
	ErrNoTenant = errors.New("no tenant in context")
	// ErrInvalidTenant is returned when tenant is invalid
	ErrInvalidTenant = errors.New("invalid tenant")
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID       string `json:"id" gorm:"primaryKey"`
	Name     string `json:"name" gorm:"not null"`
	Slug     string `json:"slug" gorm:"uniqueIndex;not null"`
	Domain   string `json:"domain,omitempty" gorm:"uniqueIndex"`
	Active   bool   `json:"active" gorm:"default:true"`
	Settings map[string]interface{} `json:"settings,omitempty" gorm:"type:jsonb"`
	Metadata map[string]string `json:"metadata,omitempty" gorm:"type:jsonb"`
}

// TenantAware is an interface for models that are tenant-scoped
type TenantAware interface {
	GetTenantID() string
	SetTenantID(tenantID string)
}

// BaseTenantModel can be embedded in models to make them tenant-aware
type BaseTenantModel struct {
	TenantID string `json:"tenant_id" gorm:"index;not null"`
}

// GetTenantID returns the tenant ID
func (m *BaseTenantModel) GetTenantID() string {
	return m.TenantID
}

// SetTenantID sets the tenant ID
func (m *BaseTenantModel) SetTenantID(tenantID string) {
	m.TenantID = tenantID
}

type tenantKey struct{}

// Config holds multi-tenancy configuration
type Config struct {
	// Strategy defines how tenant is identified
	// Options: "header", "subdomain", "path", "custom"
	Strategy string
	// HeaderName for header strategy (default: "X-Tenant-ID")
	HeaderName string
	// PathPrefix for path strategy (e.g., "/tenants/:tenant_id")
	PathPrefix string
	// RequireTenant determines if tenant is required
	RequireTenant bool
	// OnTenantNotFound callback when tenant not found
	OnTenantNotFound func(c *gin.Context)
}

// DefaultConfig returns default multi-tenancy configuration
func DefaultConfig() Config {
	return Config{
		Strategy:      "header",
		HeaderName:    "X-Tenant-ID",
		RequireTenant: true,
	}
}

// Manager manages tenants
type Manager struct {
	tenants map[string]*Tenant
	mutex   sync.RWMutex
	config  Config
}

// NewManager creates a new tenant manager
func NewManager(config Config) *Manager {
	return &Manager{
		tenants: make(map[string]*Tenant),
		config:  config,
	}
}

// AddTenant adds a tenant
func (m *Manager) AddTenant(tenant *Tenant) error {
	if tenant.ID == "" {
		return ErrInvalidTenant
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.tenants[tenant.ID] = tenant
	return nil
}

// GetTenant retrieves a tenant by ID
func (m *Manager) GetTenant(tenantID string) (*Tenant, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return nil, ErrTenantNotFound
	}

	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (m *Manager) GetTenantBySlug(slug string) (*Tenant, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, tenant := range m.tenants {
		if tenant.Slug == slug {
			return tenant, nil
		}
	}

	return nil, ErrTenantNotFound
}

// GetTenantByDomain retrieves a tenant by domain
func (m *Manager) GetTenantByDomain(domain string) (*Tenant, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, tenant := range m.tenants {
		if tenant.Domain == domain {
			return tenant, nil
		}
	}

	return nil, ErrTenantNotFound
}

// GetAllTenants returns all tenants
func (m *Manager) GetAllTenants() []*Tenant {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tenants := make([]*Tenant, 0, len(m.tenants))
	for _, tenant := range m.tenants {
		tenants = append(tenants, tenant)
	}

	return tenants
}

// RemoveTenant removes a tenant
func (m *Manager) RemoveTenant(tenantID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.tenants, tenantID)
	return nil
}

// Middleware returns a Gin middleware for tenant identification
func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string
		var err error

		switch m.config.Strategy {
		case "header":
			tenantID = c.GetHeader(m.config.HeaderName)
		case "subdomain":
			tenantID, err = extractSubdomain(c.Request.Host)
		case "path":
			tenantID = c.Param("tenant_id")
		case "custom":
			// Custom logic can be implemented via callback
			tenantID = c.GetString("tenant_id")
		default:
			tenantID = c.GetHeader("X-Tenant-ID")
		}

		if err != nil || tenantID == "" {
			if m.config.RequireTenant {
				if m.config.OnTenantNotFound != nil {
					m.config.OnTenantNotFound(c)
				} else {
					c.JSON(400, gin.H{"error": "Tenant not found"})
				}
				c.Abort()
				return
			}
		}

		if tenantID != "" {
			// Validate tenant exists
			tenant, err := m.GetTenant(tenantID)
			if err != nil && m.config.RequireTenant {
				c.JSON(404, gin.H{"error": "Tenant not found"})
				c.Abort()
				return
			}

			if tenant != nil && !tenant.Active {
				c.JSON(403, gin.H{"error": "Tenant is inactive"})
				c.Abort()
				return
			}

			// Store in context
			ctx := context.WithValue(c.Request.Context(), tenantKey{}, tenantID)
			c.Request = c.Request.WithContext(ctx)
			c.Set("tenant_id", tenantID)

			if tenant != nil {
				c.Set("tenant", tenant)
			}
		}

		c.Next()
	}
}

// extractSubdomain extracts subdomain from host
func extractSubdomain(host string) (string, error) {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", errors.New("no subdomain found")
	}

	return parts[0], nil
}

// FromContext retrieves tenant ID from context
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantKey{}).(string)
	if !ok || tenantID == "" {
		return "", ErrNoTenant
	}

	return tenantID, nil
}

// MustGetTenantID retrieves tenant ID from context or panics
func MustGetTenantID(ctx context.Context) string {
	tenantID, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}

	return tenantID
}

// TenantScope applies tenant filtering to GORM queries
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID, err := FromContext(ctx)
		if err != nil {
			return db
		}

		return db.Where("tenant_id = ?", tenantID)
	}
}

// CreateCallback is a GORM callback to auto-set tenant ID on create
func CreateCallback(db *gorm.DB) {
	if db.Statement.Schema != nil {
		tenantID, err := FromContext(db.Statement.Context)
		if err != nil {
			return
		}

		// Set tenant_id field if model is tenant-aware
		if field := db.Statement.Schema.LookUpField("tenant_id"); field != nil {
			field.Set(db.Statement.ReflectValue, tenantID)
		}
	}
}

// QueryCallback is a GORM callback to auto-filter by tenant ID
func QueryCallback(db *gorm.DB) {
	if db.Statement.Schema != nil {
		if field := db.Statement.Schema.LookUpField("tenant_id"); field != nil {
			tenantID, err := FromContext(db.Statement.Context)
			if err != nil {
				return
			}

			db.Statement.AddClause(clause.Where{
				Exprs: []clause.Expression{
					clause.Eq{
						Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"},
						Value:  tenantID,
					},
				},
			})
		}
	}
}

// RegisterCallbacks registers GORM callbacks for automatic tenant scoping
func RegisterCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("multitenancy:create", CreateCallback)
	db.Callback().Query().Before("gorm:query").Register("multitenancy:query", QueryCallback)
	db.Callback().Update().Before("gorm:update").Register("multitenancy:update", QueryCallback)
	db.Callback().Delete().Before("gorm:delete").Register("multitenancy:delete", QueryCallback)
}

// TenantDB wraps GORM DB with automatic tenant scoping
type TenantDB struct {
	*gorm.DB
	ctx context.Context
}

// NewTenantDB creates a new tenant-scoped database
func NewTenantDB(db *gorm.DB, ctx context.Context) *TenantDB {
	tenantID, err := FromContext(ctx)
	if err != nil {
		return &TenantDB{DB: db, ctx: ctx}
	}

	return &TenantDB{
		DB:  db.Where("tenant_id = ?", tenantID),
		ctx: ctx,
	}
}

// WithContext returns a new TenantDB with the given context
func (tdb *TenantDB) WithContext(ctx context.Context) *TenantDB {
	return NewTenantDB(tdb.DB, ctx)
}

// Example model with tenant isolation:
//
// type Order struct {
//     ID          string    `json:"id" gorm:"primaryKey"`
//     TenantID    string    `json:"tenant_id" gorm:"index;not null"`
//     CustomerID  string    `json:"customer_id"`
//     Total       float64   `json:"total"`
//     Status      string    `json:"status"`
//     CreatedAt   time.Time `json:"created_at"`
// }
//
// func (o *Order) GetTenantID() string { return o.TenantID }
// func (o *Order) SetTenantID(id string) { o.TenantID = id }

// Example usage:
//
// // Initialize tenant manager
// manager := multitenancy.NewManager(multitenancy.Config{
//     Strategy:      "header",
//     HeaderName:    "X-Tenant-ID",
//     RequireTenant: true,
// })
//
// // Add tenants
// manager.AddTenant(&multitenancy.Tenant{
//     ID:     "tenant-1",
//     Name:   "Acme Corp",
//     Slug:   "acme",
//     Domain: "acme.example.com",
//     Active: true,
// })
//
// manager.AddTenant(&multitenancy.Tenant{
//     ID:     "tenant-2",
//     Name:   "Beta Inc",
//     Slug:   "beta",
//     Domain: "beta.example.com",
//     Active: true,
// })
//
// // Setup middleware
// router := gin.New()
// router.Use(manager.Middleware())
//
// // Register GORM callbacks for automatic tenant scoping
// multitenancy.RegisterCallbacks(db)
//
// // In handlers, data is automatically scoped to tenant
// router.GET("/orders", func(c *gin.Context) {
//     ctx := c.Request.Context()
//     tenantID, _ := multitenancy.FromContext(ctx)
//
//     var orders []Order
//     // Automatically filtered by tenant_id
//     db.WithContext(ctx).Find(&orders)
//
//     c.JSON(200, orders)
// })
//
// // Manual tenant scoping
// router.GET("/orders/:id", func(c *gin.Context) {
//     ctx := c.Request.Context()
//
//     var order Order
//     db.WithContext(ctx).
//         Scopes(multitenancy.TenantScope(ctx)).
//         First(&order, "id = ?", c.Param("id"))
//
//     c.JSON(200, order)
// })
//
// // Using TenantDB wrapper
// router.POST("/orders", func(c *gin.Context) {
//     ctx := c.Request.Context()
//     tdb := multitenancy.NewTenantDB(db, ctx)
//
//     var order Order
//     c.BindJSON(&order)
//
//     // Tenant ID automatically set
//     tdb.Create(&order)
//
//     c.JSON(201, order)
// })
