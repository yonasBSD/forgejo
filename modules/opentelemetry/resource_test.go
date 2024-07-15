// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestResourceServiceName(t *testing.T) {
	ctx := context.Background()

	resource, err := newResource(ctx)
	assert.NoError(t, err)
	serviceKeyPresent := false
	for _, v := range resource.Attributes() {
		if v.Key == semconv.ServiceNameKey {
			assert.Equal(t, "forgejo", v.Value.AsString())
			serviceKeyPresent = true
		}
	}
	assert.True(t, serviceKeyPresent)
	serviceKeyPresent = false
	defer test.MockVariableValue(&setting.OpenTelemetry.ServiceName, "non-default value")()
	resource, err = newResource(ctx)
	assert.NoError(t, err)
	for _, v := range resource.Attributes() {
		if v.Key == semconv.ServiceNameKey {
			assert.Equal(t, "non-default value", v.Value.AsString())
			serviceKeyPresent = true
		}
	}
	assert.True(t, serviceKeyPresent)
}

func TestResourceAttributes(t *testing.T) {
	ctx := context.Background()
	defer test.MockVariableValue(&setting.OpenTelemetry.ResourceDetectors, "foo")()
	defer test.MockProtect(&setting.OpenTelemetry.ResourceAttributes)()
	setting.OpenTelemetry.ResourceAttributes = "Test=LABEL,broken,unescape=%XXlabel"
	res, err := newResource(ctx)
	assert.NoError(t, err)
	expected, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(setting.OpenTelemetry.ServiceName),
		semconv.ServiceVersion(setting.ForgejoVersion),
		attribute.String("Test", "LABEL"),
		attribute.String("unescape", "%XXlabel"),
	))
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDecoderParity(t *testing.T) {
	ctx := context.Background()
	defer test.MockVariableValue(&setting.OpenTelemetry.ResourceDetectors, "sdk,process,os,host")()
	exp, err := resource.New(
		ctx, resource.WithTelemetrySDK(), resource.WithOS(), resource.WithProcess(), resource.WithHost(), resource.WithAttributes(
			semconv.ServiceName(setting.OpenTelemetry.ServiceName), semconv.ServiceVersion(setting.ForgejoVersion),
		),
	)
	assert.NoError(t, err)
	res2, err := newResource(ctx)
	assert.NoError(t, err)
	assert.Equal(t, exp, res2)
}
