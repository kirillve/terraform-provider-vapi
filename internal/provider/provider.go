// Package provider composes the vapi Terraform provider.
//
// In scope: provider metadata, configuration schema (url, token), registration of resource
// sets, and the black-box provider-protocol characterization suite.
// Out of scope: resource models, API payload conversion, API transports.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/kirillve/terraform-provider-vapi/internal/providerdata"
	"github.com/kirillve/terraform-provider-vapi/internal/vapiv1"
)

// Ensure VAPIProvider satisfies various provider interfaces.
var _ provider.Provider = &VAPIProvider{}
var _ provider.ProviderWithFunctions = &VAPIProvider{}

// VAPIProvider defines the provider implementation.
type VAPIProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type ScaffoldingProviderModel struct {
	URL   string `tfsdk:"url"`
	Token string `tfsdk:"token"`
}

// Metadata sets the provider type name and version.
func (p *VAPIProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vapi"
	resp.Version = p.version
}

// Schema defines the provider configuration attributes, url and token.
func (p *VAPIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "The base URL of the remote API.",
				Required:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Bearer token used for API authentication.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure reads url and token and publishes them as a providerdata.Config to resources and data sources.
func (p *VAPIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	cfg := providerdata.Config{
		URL:   data.URL,
		Token: data.Token,
	}

	resp.DataSourceData = cfg
	resp.ResourceData = cfg
}

// Resources registers the v1 resource set.
func (p *VAPIProvider) Resources(_ context.Context) []func() resource.Resource {
	return vapiv1.Resources()
}

// DataSources registers no data sources.
func (p *VAPIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Functions registers no provider functions.
func (p *VAPIProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{}
}

// New returns the provider factory for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VAPIProvider{
			version: version,
		}
	}
}
