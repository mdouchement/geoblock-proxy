package main

// Based on https://github.com/mdouchement/geoblock

// Rule data types.
const (
	RuleTypeCountry RuleType = "country"
	RuleTypeCIDR    RuleType = "cidr"
)

// Supported default actions.
const (
	DefaultActionAllow = "allow"
	DefaultActionBlock = "block"
)

type (
	// A Configuration defines the proxy configuration.
	Configuration struct {
		Endpoints     []string `yaml:"endpoints"`
		Metrics       string   `yaml:"metrics"`
		Logger        string   `yaml:"logger"`
		Databases     []string `yaml:"databases"`      // Path to ip2location database files.
		DefaultAction string   `yaml:"default_action"` // Default action to perform when there is no specified rule.
		Allowlist     []Rule   `yaml:"allowlist"`
		Blocklist     []Rule   `yaml:"blocklist"`
	}

	// A RuleType defines the type of a rule.
	RuleType string

	// A Rule is used to define if a request can be allowed or blocked.
	Rule struct {
		Type  RuleType
		Value string
	}
)
