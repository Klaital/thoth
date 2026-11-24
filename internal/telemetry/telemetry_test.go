package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSetupDisabledWithoutEndpoint(t *testing.T) {
	for _, k := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	} {
		t.Setenv(k, "")
	}
	if Enabled() {
		t.Fatal("Enabled() = true with no endpoint")
	}
	shutdown, err := Setup(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSDKDisabledOverridesEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	if Enabled() {
		t.Fatal("Enabled() = true with OTEL_SDK_DISABLED=true")
	}
}

// Instruments are created from the global meter at init; they must forward to
// a provider installed later, as Setup does.
func TestBusinessMetricsReachLaterProvider(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	ctx := context.Background()
	RecordLogin(ctx, "password", nil)
	RecordLogin(ctx, "password", errors.New("bad password"))
	RecordLogin(ctx, "password", nil)
	if err := RegisterUsageGauges(func(context.Context) (UsageTotals, error) {
		return UsageTotals{Users: 3, Cards: 42}, nil
	}); err != nil {
		t.Fatal(err)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}

	logins := map[string]int64{}
	gauges := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				if m.Name == "thoth.auth.logins" {
					for _, dp := range d.DataPoints {
						v, _ := dp.Attributes.Value(attribute.Key("result"))
						logins[v.AsString()] = dp.Value
					}
				}
			case metricdata.Gauge[int64]:
				gauges[m.Name] = d.DataPoints[0].Value
			}
		}
	}
	if logins["ok"] != 2 || logins["fail"] != 1 {
		t.Errorf("logins = %v, want ok=2 fail=1", logins)
	}
	if gauges["thoth.users.total"] != 3 || gauges["thoth.cards.total"] != 42 {
		t.Errorf("gauges = %v", gauges)
	}
}
