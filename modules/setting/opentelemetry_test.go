// // Copyright 2024 The Forgejo Authors. All rights reserved.
// // SPDX-License-Identifier: MIT

package setting

import (
	"testing"
	"time"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOpenTelemetryConfiguration(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	EXPORTER_OTLP_ENDPOINT = http://jaeger:4317/
	EXPORTER_OTLP_TIMEOUT = 30s
	EXPORTER_OTLP_COMPRESSION = gzip
	EXPORTER_OTLP_INSECURE = TRUE
	EXPORTER_OTLP_HEADERS=foo=bar,overwrite=false
	SERVICE_NAME = test service
	RESOURCE_ATTRIBUTES = foo=bar
	TRACES_SAMPLER = always_on
	EXPORTER_OTLP_TRACES_TIMEOUT=5s
	EXPORTER_OTLP_TRACES_HEADERS=overwrite=true,foobar=barfoo
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "test service", OpenTelemetry.Resource.ServiceName)
	assert.Equal(t, "foo=bar", OpenTelemetry.Resource.Attributes)
	assert.Equal(t, 5*time.Second, OpenTelemetry.Traces.Timeout)
	assert.Equal(t, "gzip", OpenTelemetry.Traces.Compression)
	assert.Equal(t, sdktrace.AlwaysSample(), OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foo")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foo"], "bar")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foobar")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foobar"], "barfoo")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "overwrite")
	assert.Equal(t, OpenTelemetry.Traces.Headers["overwrite"], "true")
}

func TestOpenTelemetryTraceDisable(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	EXPORTER_OTLP_ENDPOINT = http://jaeger:4317/
	EXPORTER_OTLP_TRACES_ENDPOINT =
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.False(t, IsOpenTelemetryEnabled())
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)
}

func TestSamplerCombinations(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	type config struct {
		IniCfg   string
		Expected sdktrace.Sampler
	}
	testSamplers := []config{
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = always_on
  TRACES_SAMPLER_ARG = nothing`, sdktrace.AlwaysSample()},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = always_off`, sdktrace.NeverSample()},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = traceidratio
  TRACES_SAMPLER_ARG = 0.7`, sdktrace.TraceIDRatioBased(0.7)},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = traceidratio
  TRACES_SAMPLER_ARG = badarg`, sdktrace.TraceIDRatioBased(1)},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = parentbased_always_off`, sdktrace.ParentBased(sdktrace.NeverSample())},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = parentbased_always_of`, sdktrace.ParentBased(sdktrace.AlwaysSample())},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = parentbased_traceidratio
  TRACES_SAMPLER_ARG = 0.3`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.3))},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = parentbased_traceidratio
  TRACES_SAMPLER_ARG = badarg`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1))},
		{`[opentelemetry]
	EXPORTER_OTLP_ENDPOINT=http://localhost:4317
  TRACES_SAMPLER = not existing
  TRACES_SAMPLER_ARG = badarg`, sdktrace.ParentBased(sdktrace.AlwaysSample())},
	}

	for _, sampler := range testSamplers {
		cfg, err := NewConfigProviderFromData(sampler.IniCfg)
		assert.NoError(t, err)
		loadOpenTelemetryFrom(cfg)
		assert.Equal(t, sampler.Expected, OpenTelemetry.Traces.Sampler)
	}
}

func TestOpentelemetryBadConfigs(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := `
	[opentelemetry]
	EXPORTER_OTLP_ENDPOINT = jaeger:4317/
	`
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.False(t, IsOpenTelemetryEnabled())
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)

	iniStr = ``
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	EXPORTER_OTLP_ENDPOINT = http://jaeger:4317/

	EXPORTER_OTLP_TIMEOUT = abc
	EXPORTER_OTLP_COMPRESSION = foo
	EXPORTER_OTLP_HEADERS=%s=bar,foo=%h,foo
	SERVICE_NAME =
  TRACES_SAMPLER = not existing one
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "forgejo", OpenTelemetry.Resource.ServiceName)
	assert.Equal(t, 10*time.Second, OpenTelemetry.Traces.Timeout)
	assert.Equal(t, sdktrace.ParentBased(sdktrace.AlwaysSample()), OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Empty(t, OpenTelemetry.Traces.Headers)
}
