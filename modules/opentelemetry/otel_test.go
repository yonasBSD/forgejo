// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestNoopDefault(t *testing.T) {
	inMem := tracetest.NewInMemoryExporter()
	called := false
	exp := func(ctx context.Context) (sdktrace.SpanExporter, error) {
		called = true
		return inMem, nil
	}
	defer test.MockVariableValue(&newTraceExporter, exp)()
	ctx := context.Background()
	assert.NoError(t, Init(ctx))
	tracer := otel.Tracer("test_noop")

	_, span := tracer.Start(ctx, "test span")

	assert.False(t, span.SpanContext().HasTraceID())
	assert.False(t, span.SpanContext().HasSpanID())
	assert.False(t, called)
	graceful.GetManager().DoGracefulShutdown()
}

func TestOtelIntegration(t *testing.T) {
	const ServiceName = "forgejo-integration"
	otelURL, ok := os.LookupEnv("TEST_OTEL_URL")
	if !ok {
		t.Skip("TEST_OTEL_URL not set")
	}
	traceEndpoint, err := url.Parse(otelURL)
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, ServiceName)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, traceEndpoint)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Insecure, true)()

	ctx := context.Background()
	assert.NoError(t, Init(ctx))

	tracer := otel.Tracer("test_jaeger")
	_, span := tracer.Start(ctx, "test span")

	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	// Give the exporter time to send the span
	time.Sleep(8 * time.Second)

	traceEndpoint.Host = traceEndpoint.Hostname() + ":16686"
	traceEndpoint.Path = "/api/services"

	resp, err := http.Get(traceEndpoint.String())

	assert.NoError(t, err)

	apiResponse, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(apiResponse), ServiceName)

	graceful.GetManager().DoGracefulShutdown()
}

func TestOtelTls(t *testing.T) {
	grpcMethods := make(chan string)
	tlsConfig := generateTestTLS(t, os.TempDir(), "localhost,127.0.0.1")
	assert.NotNil(t, tlsConfig)

	collector := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)), grpc.UnknownServiceHandler(func(srv any, stream grpc.ServerStream) error {
		method, _ := grpc.Method(stream.Context())
		grpcMethods <- method
		return nil
	}))
	t.Cleanup(collector.GracefulStop)
	ln, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	t.Cleanup(func() {
		ln.Close()
	})
	go collector.Serve(ln)

	traceEndpoint, err := url.Parse("https://" + ln.Addr().String())
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, "forgejo-certs")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, traceEndpoint)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Certificate, os.TempDir()+"/"+"cert.pem")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.ClientCertificate, os.TempDir()+"/"+"cert.pem")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.ClientKey, os.TempDir()+"/"+"key.pem")()
	ctx := context.Background()
	assert.NoError(t, Init(ctx))

	tracer := otel.Tracer("test_jaeger")
	_, span := tracer.Start(ctx, "test span")
	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	// Give the exporter time to send the span
	select {
	case method := <-grpcMethods:
		assert.Equal(t, "/opentelemetry.proto.collector.trace.v1.TraceService/Export", method)
	case <-time.After(10 * time.Second):
		t.Fatal("no grpc call within 10s")
	}
	graceful.GetManager().DoGracefulShutdown()
}

func TestExporter(t *testing.T) {
	inMem := tracetest.NewInMemoryExporter()
	exp := func(ctx context.Context) (sdktrace.SpanExporter, error) {
		return inMem, nil
	}
	defer test.MockVariableValue(&newTraceExporter, exp)()

	// Force feature activation
	endpoint, err := url.Parse("http://localhost:4317")
	assert.NoError(t, err)
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, endpoint)()

	ctx := context.Background()
	assert.NoError(t, Init(ctx))
	tracer := otel.Tracer("test_grpc")

	_, span := tracer.Start(ctx, "test span")

	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())
	graceful.GetManager().DoGracefulShutdown()
}

func generateTestTLS(t *testing.T, path, host string) *tls.Config {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	keyUsage := x509.KeyUsageDigitalSignature

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Forgejo Testing"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}
	certOut, err := os.Create(path + "/cert.pem")
	if err != nil {
		t.Fatalf("Failed to open cert.pem for writing: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	if err := certOut.Close(); err != nil {
		t.Fatalf("Error closing cert.pem: %v", err)
	}
	keyOut, err := os.OpenFile(path+"/key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("Failed to open key.pem for writing: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		t.Fatalf("Error closing key.pem: %v", err)
	}
	serverCert, err := tls.LoadX509KeyPair(path+"/cert.pem", path+"/key.pem")
	if err != nil {
		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAnyClientCert,
	}
}
