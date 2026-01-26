/*
2025 © Postgres.ai
*/

package configs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOtelCollectorConfigValid(t *testing.T) {
	data, err := os.ReadFile("otel-collector.example.yml")
	require.NoError(t, err, "failed to read otel-collector.example.yml")

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err, "otel config is not valid yaml")

	assert.Contains(t, config, "receivers", "config must have receivers section")
	assert.Contains(t, config, "exporters", "config must have exporters section")
	assert.Contains(t, config, "service", "config must have service section")
}

func TestOtelCollectorConfigHasPrometheusReceiver(t *testing.T) {
	data, err := os.ReadFile("otel-collector.example.yml")
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	receivers, ok := config["receivers"].(map[string]interface{})
	require.True(t, ok, "receivers must be a map")

	assert.Contains(t, receivers, "prometheus", "must have prometheus receiver for dblab scraping")
}

func TestOtelCollectorConfigHasOTLPExporter(t *testing.T) {
	data, err := os.ReadFile("otel-collector.example.yml")
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	exporters, ok := config["exporters"].(map[string]interface{})
	require.True(t, ok, "exporters must be a map")

	assert.Contains(t, exporters, "otlp", "must have otlp exporter for otel backends")
}

func TestOtelCollectorConfigHasMetricsPipeline(t *testing.T) {
	data, err := os.ReadFile("otel-collector.example.yml")
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	service, ok := config["service"].(map[string]interface{})
	require.True(t, ok, "service must be a map")

	pipelines, ok := service["pipelines"].(map[string]interface{})
	require.True(t, ok, "service must have pipelines")

	assert.Contains(t, pipelines, "metrics", "must have metrics pipeline")
}

func TestOtelCollectorConfigDBLabScrapeTarget(t *testing.T) {
	data, err := os.ReadFile("otel-collector.example.yml")
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	receivers := config["receivers"].(map[string]interface{})
	prometheus := receivers["prometheus"].(map[string]interface{})
	promConfig := prometheus["config"].(map[string]interface{})
	scrapeConfigs := promConfig["scrape_configs"].([]interface{})

	found := false

	for _, sc := range scrapeConfigs {
		scrapeConfig := sc.(map[string]interface{})
		if scrapeConfig["job_name"] == "dblab" {
			found = true

			staticConfigs := scrapeConfig["static_configs"].([]interface{})
			require.NotEmpty(t, staticConfigs)

			firstConfig := staticConfigs[0].(map[string]interface{})
			targets := firstConfig["targets"].([]interface{})
			assert.Contains(t, targets, "localhost:2345", "must scrape dblab default port")

			break
		}
	}

	assert.True(t, found, "must have dblab scrape job")
}
