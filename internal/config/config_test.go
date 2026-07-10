package config

import "testing"

func TestLoadUsesRailwayPortWhenPresent(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("METRICS_HTTP_PORT", "")

	cfg := Load()
	if cfg.MetricsHTTPPort != "9090" {
		t.Fatalf("expected metrics port 9090, got %q", cfg.MetricsHTTPPort)
	}
}

func TestLoadUsesRailwayPublicDomainForMetricsBaseURL(t *testing.T) {
	t.Setenv("METRICS_BASE_URL", "")
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "metrics-example.up.railway.app")

	cfg := Load()
	if cfg.MetricsBaseURL != "https://metrics-example.up.railway.app" {
		t.Fatalf("expected metrics base URL https://metrics-example.up.railway.app, got %q", cfg.MetricsBaseURL)
	}
}

func TestLoadPrefersExplicitMetricsBaseURL(t *testing.T) {
	t.Setenv("METRICS_BASE_URL", "https://metrics-service.example.com")
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "ignored.up.railway.app")

	cfg := Load()
	if cfg.MetricsBaseURL != "https://metrics-service.example.com" {
		t.Fatalf("expected explicit metrics base URL to be used, got %q", cfg.MetricsBaseURL)
	}
}

func TestLoadUsesRailwayPublicDomainWhenExplicitValueIsInternalDockerHost(t *testing.T) {
	t.Setenv("METRICS_BASE_URL", "http://metrics-service:8080")
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "metrics-example.up.railway.app")

	cfg := Load()
	if cfg.MetricsBaseURL != "https://metrics-example.up.railway.app" {
		t.Fatalf("expected Railway public domain to override internal docker host, got %q", cfg.MetricsBaseURL)
	}
}
