// Package vapiv1 is the frozen v1 resource set of the vapi Terraform provider:
// vapi_assistant, vapi_file, vapi_twilio_phone_number, vapi_sip_trunk_phone_number,
// vapi_tool_function, vapi_tool_query_function and vapi_sip_trunk.
//
// In scope: schemas, models, expand/flatten, CRUD and import for those seven types,
// built on the v1 HTTP client in internal/vapi.
// Out of scope: provider configuration schema (internal/provider), any v2 resource,
// the VAPI SDK, state migration.
//
// Behaviour is frozen: a change here must leave internal/provider/testdata/v1 untouched.
// A golden diff is a user-visible change and needs an explicit sign-off and a CHANGELOG entry.
// Tier 2.
package vapiv1

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/kirillve/terraform-provider-vapi/internal/providerdata"
	"github.com/kirillve/terraform-provider-vapi/internal/vapi"
)

// Resources returns a fresh slice of the seven v1 resource constructors, in registration order.
func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		NewVAPIAssistantResource,
		NewVAPIFileResource,
		NewVAPIPhoneNumberResource,
		NewVAPIToolFunctionResource,
		NewVAPIToolQueryFunctionResource,
		NewVAPISIPTrunkResource,
		NewVAPISIPTrunkPhoneNumberResource,
	}
}

// configureClient returns the v1 API client for the providerdata.Config that the provider
// published as ProviderData. A nil ProviderData (the validate-time Configure) yields nil with
// no diagnostics; any other type yields nil and one error diagnostic.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *vapi.APIClient {
	if req.ProviderData == nil {
		return nil
	}

	cfg, ok := req.ProviderData.(providerdata.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected providerdata.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}

	return &vapi.APIClient{BaseURL: cfg.URL, Token: cfg.Token, HTTPClient: &http.Client{}}
}
