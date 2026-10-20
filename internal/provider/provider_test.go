package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkProvider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirillve/terraform-provider-vapi/internal/providerdata"
)

func TestProviderMetadataSchemaAndConfigure(t *testing.T) {
	ctx := t.Context()

	providerFactory := New("1.2.3")
	prov, ok := providerFactory().(*VAPIProvider)
	if !ok {
		t.Fatalf("expected *VAPIProvider")
	}

	metaResp := &frameworkProvider.MetadataResponse{}
	prov.Metadata(ctx, frameworkProvider.MetadataRequest{}, metaResp)
	if metaResp.TypeName != "vapi" {
		t.Fatalf("unexpected type name %s", metaResp.TypeName)
	}
	if metaResp.Version != "1.2.3" {
		t.Fatalf("unexpected version %s", metaResp.Version)
	}

	var schemaResp frameworkProvider.SchemaResponse
	prov.Schema(ctx, frameworkProvider.SchemaRequest{}, &schemaResp)
	if len(schemaResp.Schema.Attributes) != 2 {
		t.Fatalf("expected provider schema to define 2 attributes")
	}

	configValue := buildProviderConfig(ctx, t)

	confResp := &frameworkProvider.ConfigureResponse{}
	prov.Configure(ctx, frameworkProvider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw:    configValue,
			Schema: schemaResp.Schema,
		},
	}, confResp)

	if confResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %v", confResp.Diagnostics)
	}

	want := providerdata.Config{URL: "https://api.example.com", Token: "secret"}
	assert.Equal(t, want, confResp.ResourceData, "resource data must be a providerdata.Config value")
	assert.Equal(t, want, confResp.DataSourceData, "data source data must be a providerdata.Config value")

	if len(prov.Resources(ctx)) == 0 {
		t.Fatalf("expected resources to be registered")
	}
	if len(prov.DataSources(ctx)) != 0 {
		t.Fatalf("expected no data sources")
	}
	if len(prov.Functions(ctx)) != 0 {
		t.Fatalf("expected no functions")
	}
}

func TestProviderConfigure_WhenURLMissingOrTokenMissing_KeepsTodaysDiagnostics(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		url, token any
		wantPath   string
	}{
		"url null":   {url: nil, token: "secret", wantPath: "Path: url\n"},
		"token null": {url: "https://api.example.com", token: nil, wantPath: "Path: token\n"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			prov := &VAPIProvider{version: "test"}

			var schemaResp frameworkProvider.SchemaResponse
			prov.Schema(ctx, frameworkProvider.SchemaRequest{}, &schemaResp)

			raw := tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{"url": tftypes.String, "token": tftypes.String}},
				map[string]tftypes.Value{
					"url":   tftypes.NewValue(tftypes.String, tc.url),
					"token": tftypes.NewValue(tftypes.String, tc.token),
				},
			)

			resp := &frameworkProvider.ConfigureResponse{}
			prov.Configure(ctx, frameworkProvider.ConfigureRequest{Config: tfsdk.Config{Raw: raw, Schema: schemaResp.Schema}}, resp)

			require.Equal(t, 1, resp.Diagnostics.ErrorsCount())
			diagnostic := resp.Diagnostics.Errors()[0]
			assert.Equal(t, "Value Conversion Error", diagnostic.Summary())
			assert.Contains(t, diagnostic.Detail(), tc.wantPath)
			assert.Contains(t, diagnostic.Detail(), "Target Type: string")
			assert.Nil(t, resp.ResourceData)
			assert.Nil(t, resp.DataSourceData)
		})
	}
}

func buildProviderConfig(ctx context.Context, t *testing.T) tftypes.Value {
	t.Helper()

	obj, diags := types.ObjectValue(map[string]attr.Type{
		"url":   types.StringType,
		"token": types.StringType,
	}, map[string]attr.Value{
		"url":   types.StringValue("https://api.example.com"),
		"token": types.StringValue("secret"),
	})
	if diags.HasError() {
		t.Fatalf("object value diagnostics: %v", diags)
	}

	value, err := obj.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("terraform value conversion failed: %v", err)
	}

	return value
}
