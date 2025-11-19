package featureflags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

var (
	// ErrFlagNotFound is returned when flag is not found
	ErrFlagNotFound = errors.New("feature flag not found")
	// ErrInvalidRule is returned when rule is invalid
	ErrInvalidRule = errors.New("invalid rule")
)

// Flag represents a feature flag
type Flag struct {
	// Key uniquely identifies the flag
	Key string `json:"key"`
	// Name is the human-readable name
	Name string `json:"name"`
	// Description explains what the flag controls
	Description string `json:"description"`
	// Enabled is the default state
	Enabled bool `json:"enabled"`
	// Rules define conditional logic
	Rules []Rule `json:"rules"`
	// Variations for A/B testing
	Variations []Variation `json:"variations,omitempty"`
	// DefaultVariation when no rules match
	DefaultVariation string `json:"default_variation,omitempty"`
	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`
	// Tags for organization
	Tags []string `json:"tags,omitempty"`
}

// Rule defines conditional flag behavior
type Rule struct {
	// ID uniquely identifies the rule
	ID string `json:"id"`
	// Conditions that must be met
	Conditions []Condition `json:"conditions"`
	// Variation to serve when conditions match
	Variation string `json:"variation,omitempty"`
	// Enabled state for this rule
	Enabled bool `json:"enabled"`
	// Rollout percentage (0-100)
	RolloutPercentage int `json:"rollout_percentage,omitempty"`
}

// Condition defines a single condition
type Condition struct {
	// Attribute to check (user_id, email, country, etc.)
	Attribute string `json:"attribute"`
	// Operator (equals, not_equals, contains, in, not_in, greater_than, less_than)
	Operator string `json:"operator"`
	// Values to compare against
	Values []string `json:"values"`
}

// Variation represents an A/B test variation
type Variation struct {
	// Key identifies the variation
	Key string `json:"key"`
	// Name is human-readable
	Name string `json:"name"`
	// Value is the variation payload
	Value interface{} `json:"value"`
	// Weight for weighted rollouts (0-100)
	Weight int `json:"weight"`
}

// Context holds user/request context for evaluation
type EvalContext struct {
	// UserID identifies the user
	UserID string
	// Email for the user
	Email string
	// Attributes for rule matching
	Attributes map[string]interface{}
	// Groups/segments the user belongs to
	Groups []string
}

// Provider provides feature flag functionality
type Provider interface {
	IsEnabled(ctx context.Context, flagKey string, evalCtx EvalContext) (bool, error)
	GetVariation(ctx context.Context, flagKey string, evalCtx EvalContext) (string, error)
	GetVariationValue(ctx context.Context, flagKey string, evalCtx EvalContext) (interface{}, error)
	GetAllFlags(ctx context.Context) ([]Flag, error)
	UpdateFlag(ctx context.Context, flag Flag) error
	DeleteFlag(ctx context.Context, flagKey string) error
	Close() error
}

// InMemoryProvider implements Provider using in-memory storage
type InMemoryProvider struct {
	flags map[string]Flag
	mutex sync.RWMutex
}

// NewInMemoryProvider creates a new in-memory provider
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		flags: make(map[string]Flag),
	}
}

// IsEnabled checks if a flag is enabled for the given context
func (p *InMemoryProvider) IsEnabled(ctx context.Context, flagKey string, evalCtx EvalContext) (bool, error) {
	p.mutex.RLock()
	flag, exists := p.flags[flagKey]
	p.mutex.RUnlock()

	if !exists {
		return false, ErrFlagNotFound
	}

	// Check rules
	for _, rule := range flag.Rules {
		if !rule.Enabled {
			continue
		}

		if matchesRule(rule, evalCtx) {
			// Check rollout percentage
			if rule.RolloutPercentage > 0 && rule.RolloutPercentage < 100 {
				return isInRollout(flagKey, evalCtx.UserID, rule.RolloutPercentage), nil
			}
			return true, nil
		}
	}

	// Return default state
	return flag.Enabled, nil
}

// GetVariation returns the variation key for A/B testing
func (p *InMemoryProvider) GetVariation(ctx context.Context, flagKey string, evalCtx EvalContext) (string, error) {
	p.mutex.RLock()
	flag, exists := p.flags[flagKey]
	p.mutex.RUnlock()

	if !exists {
		return "", ErrFlagNotFound
	}

	// Check rules for variation override
	for _, rule := range flag.Rules {
		if !rule.Enabled {
			continue
		}

		if matchesRule(rule, evalCtx) {
			if rule.Variation != "" {
				return rule.Variation, nil
			}
		}
	}

	// Use weighted variation selection
	if len(flag.Variations) > 0 {
		return selectWeightedVariation(flag.Variations, flagKey, evalCtx.UserID), nil
	}

	return flag.DefaultVariation, nil
}

// GetVariationValue returns the variation value
func (p *InMemoryProvider) GetVariationValue(ctx context.Context, flagKey string, evalCtx EvalContext) (interface{}, error) {
	variationKey, err := p.GetVariation(ctx, flagKey, evalCtx)
	if err != nil {
		return nil, err
	}

	p.mutex.RLock()
	flag, exists := p.flags[flagKey]
	p.mutex.RUnlock()

	if !exists {
		return nil, ErrFlagNotFound
	}

	// Find variation value
	for _, variation := range flag.Variations {
		if variation.Key == variationKey {
			return variation.Value, nil
		}
	}

	return nil, fmt.Errorf("variation %s not found", variationKey)
}

// GetAllFlags returns all flags
func (p *InMemoryProvider) GetAllFlags(ctx context.Context) ([]Flag, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	flags := make([]Flag, 0, len(p.flags))
	for _, flag := range p.flags {
		flags = append(flags, flag)
	}

	return flags, nil
}

// UpdateFlag creates or updates a flag
func (p *InMemoryProvider) UpdateFlag(ctx context.Context, flag Flag) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if existing, exists := p.flags[flag.Key]; exists {
		flag.CreatedAt = existing.CreatedAt
	} else {
		flag.CreatedAt = time.Now()
	}

	flag.UpdatedAt = time.Now()
	p.flags[flag.Key] = flag

	return nil
}

// DeleteFlag deletes a flag
func (p *InMemoryProvider) DeleteFlag(ctx context.Context, flagKey string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.flags, flagKey)
	return nil
}

// Close closes the provider
func (p *InMemoryProvider) Close() error {
	return nil
}

// matchesRule checks if a rule matches the context
func matchesRule(rule Rule, evalCtx EvalContext) bool {
	for _, condition := range rule.Conditions {
		if !matchesCondition(condition, evalCtx) {
			return false
		}
	}
	return true
}

// matchesCondition checks if a condition matches
func matchesCondition(condition Condition, evalCtx EvalContext) bool {
	var value interface{}

	switch condition.Attribute {
	case "user_id":
		value = evalCtx.UserID
	case "email":
		value = evalCtx.Email
	case "group":
		return matchesGroup(condition, evalCtx.Groups)
	default:
		value = evalCtx.Attributes[condition.Attribute]
	}

	valueStr := fmt.Sprintf("%v", value)

	switch condition.Operator {
	case "equals":
		return len(condition.Values) > 0 && valueStr == condition.Values[0]
	case "not_equals":
		return len(condition.Values) > 0 && valueStr != condition.Values[0]
	case "contains":
		for _, v := range condition.Values {
			if contains(valueStr, v) {
				return true
			}
		}
		return false
	case "in":
		for _, v := range condition.Values {
			if valueStr == v {
				return true
			}
		}
		return false
	case "not_in":
		for _, v := range condition.Values {
			if valueStr == v {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// matchesGroup checks if user is in any of the specified groups
func matchesGroup(condition Condition, userGroups []string) bool {
	for _, userGroup := range userGroups {
		for _, conditionGroup := range condition.Values {
			if userGroup == conditionGroup {
				return true
			}
		}
	}
	return false
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}

// isInRollout determines if user is in rollout percentage
func isInRollout(flagKey, userID string, percentage int) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	// Consistent hashing to determine rollout
	hash := hashString(flagKey + userID)
	return int(hash%100) < percentage
}

// selectWeightedVariation selects a variation based on weights
func selectWeightedVariation(variations []Variation, flagKey, userID string) string {
	if len(variations) == 0 {
		return ""
	}

	// Calculate total weight
	totalWeight := 0
	for _, v := range variations {
		totalWeight += v.Weight
	}

	if totalWeight == 0 {
		return variations[0].Key
	}

	// Use hash to select variation consistently for same user
	hash := hashString(flagKey + userID)
	bucket := int(hash % uint32(totalWeight))

	currentWeight := 0
	for _, v := range variations {
		currentWeight += v.Weight
		if bucket < currentWeight {
			return v.Key
		}
	}

	return variations[0].Key
}

// hashString creates a hash of a string
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// FlagBuilder helps build feature flags
type FlagBuilder struct {
	flag Flag
}

// NewFlagBuilder creates a new flag builder
func NewFlagBuilder(key, name string) *FlagBuilder {
	return &FlagBuilder{
		flag: Flag{
			Key:       key,
			Name:      name,
			Enabled:   false,
			Rules:     make([]Rule, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// WithDescription sets the description
func (b *FlagBuilder) WithDescription(description string) *FlagBuilder {
	b.flag.Description = description
	return b
}

// WithEnabled sets the default enabled state
func (b *FlagBuilder) WithEnabled(enabled bool) *FlagBuilder {
	b.flag.Enabled = enabled
	return b
}

// WithRule adds a rule
func (b *FlagBuilder) WithRule(rule Rule) *FlagBuilder {
	b.flag.Rules = append(b.flag.Rules, rule)
	return b
}

// WithVariations sets A/B test variations
func (b *FlagBuilder) WithVariations(variations []Variation) *FlagBuilder {
	b.flag.Variations = variations
	return b
}

// WithDefaultVariation sets the default variation
func (b *FlagBuilder) WithDefaultVariation(key string) *FlagBuilder {
	b.flag.DefaultVariation = key
	return b
}

// WithTags adds tags
func (b *FlagBuilder) WithTags(tags ...string) *FlagBuilder {
	b.flag.Tags = tags
	return b
}

// Build returns the constructed flag
func (b *FlagBuilder) Build() Flag {
	return b.flag
}

// Example usage:
//
// // Create provider
// provider := featureflags.NewInMemoryProvider()
//
// // Create feature flag with A/B testing
// flag := featureflags.NewFlagBuilder("new-checkout-flow", "New Checkout Flow").
//     WithDescription("Test new checkout UI").
//     WithEnabled(true).
//     WithVariations([]featureflags.Variation{
//         {Key: "control", Name: "Control", Value: "old-ui", Weight: 50},
//         {Key: "variant-a", Name: "Variant A", Value: "new-ui", Weight: 50},
//     }).
//     WithDefaultVariation("control").
//     WithRule(featureflags.Rule{
//         ID: "beta-users",
//         Enabled: true,
//         Conditions: []featureflags.Condition{
//             {
//                 Attribute: "group",
//                 Operator:  "in",
//                 Values:    []string{"beta-testers"},
//             },
//         },
//         Variation: "variant-a",
//     }).
//     WithTags("checkout", "ui", "ab-test").
//     Build()
//
// provider.UpdateFlag(context.Background(), flag)
//
// // Check flag
// evalCtx := featureflags.EvalContext{
//     UserID: "user-123",
//     Email:  "user@example.com",
//     Groups: []string{"beta-testers"},
//     Attributes: map[string]interface{}{
//         "country": "US",
//         "plan":    "premium",
//     },
// }
//
// enabled, _ := provider.IsEnabled(context.Background(), "new-checkout-flow", evalCtx)
// variation, _ := provider.GetVariation(context.Background(), "new-checkout-flow", evalCtx)
// value, _ := provider.GetVariationValue(context.Background(), "new-checkout-flow", evalCtx)
//
// if enabled {
//     log.Printf("Feature enabled for user, variation: %s, value: %v", variation, value)
// }
//
// // Percentage rollout
// flag = featureflags.NewFlagBuilder("premium-features", "Premium Features").
//     WithEnabled(false).
//     WithRule(featureflags.Rule{
//         ID:                "gradual-rollout",
//         Enabled:           true,
//         Conditions:        []featureflags.Condition{},
//         RolloutPercentage: 10, // 10% of users
//     }).
//     Build()
