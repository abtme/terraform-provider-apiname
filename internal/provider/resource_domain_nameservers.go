package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/abtme/terraform-provider-apiname/internal/client"
)

var _ resource.Resource = &DomainNameserversResource{}
var _ resource.ResourceWithImportState = &DomainNameserversResource{}

type DomainNameserversResource struct {
	client *client.Client
}

type DomainNameserversModel struct {
	ID          types.String `tfsdk:"id"`
	DomainName  types.String `tfsdk:"domain_name"`
	Nameservers types.Set    `tfsdk:"nameservers"`
}

func NewDomainNameserversResource() resource.Resource {
	return &DomainNameserversResource{}
}

func (r *DomainNameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_nameservers"
}

func (r *DomainNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the nameservers for a DomainNameAPI domain registration. " +
			"This resource updates the nameservers on an existing domain — it does not register or delete domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The domain name (used as the resource identifier).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to manage, e.g. \"abt.me\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nameservers": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				// A Set, not a List - DomainNameAPI's registrar API doesn't
				// preserve submitted order on read-back (observed returning
				// nameservers in a different, but internally consistent,
				// order than whatever was sent). Nameserver order has no
				// DNS-functional meaning, so modeling this as an
				// order-sensitive List caused a permanent phantom diff on
				// every plan even when the actual nameserver set was
				// unchanged.
				Description: "Set of nameservers to assign to the domain. Order does not matter.",
			},
		},
	}
}

func (r *DomainNameserversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *DomainNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainNameserversModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns, diags := toStringSlice(ctx, plan.Nameservers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.DomainName.ValueString()
	if err := r.client.ModifyNameservers(domain, ns); err != nil {
		resp.Diagnostics.AddError("Failed to set nameservers", err.Error())
		return
	}

	plan.ID = types.StringValue(domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainNameserversModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns, err := r.client.GetNameservers(state.DomainName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read nameservers", err.Error())
		return
	}

	nsSet, diags := types.SetValueFrom(ctx, types.StringType, ns)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Nameservers = nsSet
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainNameserversModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns, diags := toStringSlice(ctx, plan.Nameservers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.DomainName.ValueString()
	if err := r.client.ModifyNameservers(domain, ns); err != nil {
		resp.Diagnostics.AddError("Failed to update nameservers", err.Error())
		return
	}

	plan.ID = types.StringValue(domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainNameserversResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Nameservers cannot be "deleted" — removing this resource from Terraform
	// state simply stops managing them. No API call is made.
}

func (r *DomainNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by domain name (e.g. terraform import apiname_domain_nameservers.this abt.me)
	domain := req.ID

	ns, err := r.client.GetNameservers(domain)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read nameservers during import", err.Error())
		return
	}

	nsSet, diags := types.SetValueFrom(ctx, types.StringType, ns)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := DomainNameserversModel{
		ID:          types.StringValue(domain),
		DomainName:  types.StringValue(domain),
		Nameservers: nsSet,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func toStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var elems []types.String
	diags := set.ElementsAs(ctx, &elems, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]string, len(elems))
	for i, e := range elems {
		result[i] = e.ValueString()
	}
	return result, diags
}
