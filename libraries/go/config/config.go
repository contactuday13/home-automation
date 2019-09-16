package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jakewright/home-automation/libraries/go/slog"
)

type Provider interface {
	Has(string) bool
	Get(string) Value
}

// Config holds a nested map of config values and provides
// helper functions for easier access and type casting.
type Config struct {
	m map[string]interface{}
	l sync.RWMutex
}

// Value is returned from Get and has
// receiver methods for casting to various types.
type Value struct {
	raw interface{}
}

var DefaultProvider Provider

func mustGetDefaultProvider() Provider {
	if DefaultProvider == nil {
		slog.Panic("Config read before default provider set")
	}

	return DefaultProvider
}

func Has(path string) bool  { return mustGetDefaultProvider().Has(path) }
func Get(path string) Value { return mustGetDefaultProvider().Get(path) }

func New(content map[string]interface{}) *Config {
	return &Config{
		m: content,
	}
}

// Replace swaps the internal config map and returns whether true if anything changed
func (c *Config) Replace(content map[string]interface{}) {
	c.l.Lock()
	defer c.l.Unlock()

	if !reflect.DeepEqual(content, c.m) {
		slog.Info("Config updated")
	}

	c.m = content
}

// Has returns whether the config has a raw at the given path e.g. "redis.host"
func (c *Config) Has(path string) bool {
	v := c.Get(path)
	return v.raw != nil
}

// Get returns the raw at the given path e.g. "redis.host"
func (c *Config) Get(path string) Value {
	c.l.RLock()
	defer c.l.RUnlock()

	return Value{
		raw: reduce(strings.Split(path, "."), c.m),
	}
}

func reduce(parts []string, value interface{}) interface{} {
	// If this is the last part of the key
	if len(parts) == 0 {
		return value
	}

	// If raw is not a map then we can't continue
	valueMap, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	// If the key we are searching for is not defined
	value, ok = valueMap[parts[0]]
	if !ok {
		return nil
	}

	return reduce(parts[1:], value)
}

// Int converts the raw to an int and panics if it cannot be represented.
// The first default is returned if raw is not defined.
func (v Value) Int(defaults ...int) int {
	// Return the default if the raw is undefined
	if v.raw == nil {
		// Make sure there's at least one thing in the list
		defaults = append(defaults, 0)
		return defaults[0]
	}

	switch t := v.raw.(type) {
	case int:
		return t
	case float64:
		if t != float64(int(t)) {
			slog.Panic("%v cannot be represented as an int", t)
		}

		return int(t)
	case string:
		i, err := strconv.Atoi(t)
		if err != nil {
			slog.Panic("Failed to convert string to int: %v", err)
		}
		return i
	default:
		slog.Panic("%v is of unsupported type %v", t, reflect.TypeOf(t).String())
	}

	return 0 // Never hit
}

// String converts the raw to a string. The first default is returned if raw is not defined.
func (v Value) String(defaults ...string) string {
	if v.raw == nil {
		defaults = append(defaults, "")
		return defaults[0]
	}

	return fmt.Sprintf("%s", v.raw)
}

// Bool converts the raw to a bool and panics if it cannot be represented.
// The first default is returned if raw is not defined.
func (v Value) Bool(defaults ...bool) bool {
	// Return the first default if the raw is undefined
	if v.raw == nil {
		// Make sure there's at least one thing in the list
		defaults = append(defaults, false)
		return defaults[0]
	}

	switch t := v.raw.(type) {
	case string:
		b, err := strconv.ParseBool(t)
		if err != nil {
			slog.Panic("Failed to parse bool: %v", err)
		}
		return b

	case bool:
		return t

	default:
		slog.Panic("%v is of unsupported type %v", t, reflect.TypeOf(t).String())
	}

	return false
}

func (v Value) Duration(defaults ...time.Duration) time.Duration {
	// Return the first default if raw is undefined
	if v.raw == nil {
		// Make sure there's at least one thing in the list
		defaults = append(defaults, 0)
		return defaults[0]
	}

	switch t := v.raw.(type) {
	case string:
		d, err := time.ParseDuration(t)
		if err != nil {
			slog.Panic("Failed to parse duration: %v", err)
		}
		return d
	default:
		slog.Panic("%v is of unsupported type %v", t, reflect.TypeOf(t).String())
	}

	return 0
}
