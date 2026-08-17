package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/abtme/terraform-provider-apiname/internal/client"
)

const defaultBaseURL = "https://api.domainresellerapi.com/api/v1"

var _ provider.Provider = &ApinameProvider{}

type ApinameProvider struct {
	version string
}

type ApinameProviderModel struct {
	BaseURL    types.String `tfsdk:"base_url"`
	ResellerID types.String `tfsdk:"reseller_id"`
	APIKey     types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ApinameProvider{version: version}
	}
}

func (p *ApinameProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "apiname"
	resp.Version = p.version
}

func (p *ApinameProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for managing DomainNameAPI (\"ApiName\") reseller domain resources.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "DomainNameAPI base URL. Defaults to https://api.domainresellerapi.com/api/v1.",
			},
			"reseller_id": schema.StringAttribute{
				Optional:    true,
				Description: "DomainNameAPI reseller ID (UUID). Can also be set via APINAME_RESELLER_ID environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "DomainNameAPI API key. Can also be set via APINAME_API_KEY environment variable.",
			},
		},
	}
}

func (p *ApinameProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ApinameProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := defaultBaseURL
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}

	resellerID := os.Getenv("APINAME_RESELLER_ID")
	if !config.ResellerID.IsNull() && !config.ResellerID.IsUnknown() {
		resellerID = config.ResellerID.ValueString()
	}
	if resellerID == "" {
		resp.Diagnostics.AddError("Missing reseller_id", "reseller_id must be set in provider config or APINAME_RESELLER_ID environment variable.")
		return
	}

	apiKey := os.Getenv("APINAME_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing api_key", "api_key must be set in provider config or APINAME_API_KEY environment variable.")
		return
	}

	c := client.NewClient(baseURL, resellerID, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *ApinameProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainNameserversResource,
	}
}

func (p *ApinameProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
