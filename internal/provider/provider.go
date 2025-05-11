package provider

import (
	"context"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func (p *VAPIProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vapi"
	resp.Version = p.version
}

func (p *VAPIProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
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

func (p *VAPIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := &vapi.APIClient{
		BaseURL:    data.URL,
		Token:      data.Token,
		HTTPClient: &http.Client{},
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *VAPIProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVAPIAssistantResource,
		NewVAPIFileResource,
		NewVAPIPhoneNumberResource,
		NewVAPIToolFunctionResource,
		NewVAPIToolQueryFunctionResource,
	}
}

func (p *VAPIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *VAPIProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VAPIProvider{
			version: version,
		}
	}
}
