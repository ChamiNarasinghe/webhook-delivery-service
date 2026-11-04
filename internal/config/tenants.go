package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Tenant struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"-"`
}

type TenantConfig struct {
	Tenants map[string]Tenant `json:"tenants"`
}

func LoadTenants(filePath string) (*TenantConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant configuration: %w", err)
	}

	var config TenantConfig

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse tenant configuration: %w", err)
	}

	if len(config.Tenants) == 0 {
		return nil, fmt.Errorf("no tenants configured")
	}

	for name, tenant := range config.Tenants {
		// Read the tenant's secret from an environment variable.
		envName := "WEBHOOK_SECRET_" + strings.ToUpper(name)
		secret := os.Getenv(envName)

		if tenant.WebhookURL == "" {
			return nil, fmt.Errorf("tenant %q has a missing webhook URL", name)
		}

		if secret == "" {
			return nil, fmt.Errorf("missing environment variable %s", envName)
		}

		tenant.Secret = secret
		config.Tenants[name] = tenant
	}

	return &config, nil
}
