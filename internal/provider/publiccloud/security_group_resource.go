package publiccloud

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/leaseweb/leaseweb-go-sdk/publiccloud"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &securityGroupResource{}
	_ resource.ResourceWithImportState = &securityGroupResource{}
)

type securityGroupResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Default types.Bool   `tfsdk:"default"`
	State   types.String `tfsdk:"state"`
}

func adaptSecurityGroupToResource(
	sdkSG publiccloud.SecurityGroup,
) *securityGroupResourceModel {
	return &securityGroupResourceModel{
		ID:      basetypes.NewStringValue(sdkSG.GetId()),
		Name:    basetypes.NewStringValue(sdkSG.GetName()),
		Default: basetypes.NewBoolValue(sdkSG.GetDefault()),
		State:   basetypes.NewStringValue(string(sdkSG.GetState())),
	}
}

type securityGroupResource struct {
	utils.ResourceAPI
}

func (s *securityGroupResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(
		ctx,
		path.Root("id"),
		request,
		response,
	)
}

func (s *securityGroupResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	response *resource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: utils.BetaDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the security group",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is the default security group",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the security group",
			},
		},
	}
}

func (s *securityGroupResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan securityGroupResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewCreateSecurityGroupOpts(plan.Name.ValueString())

	sdkSG, httpResponse, err := s.PubliccloudAPI.CreateSecurityGroup(ctx).
		CreateSecurityGroupOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptSecurityGroupToResource(*sdkSG)
	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (s *securityGroupResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state securityGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkSG, httpResponse, err := s.PubliccloudAPI.
		GetSecurityGroup(ctx, state.ID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptSecurityGroupToResource(*sdkSG)
	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (s *securityGroupResource) Update(
	ctx context.Context,
	request resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	var plan securityGroupResourceModel
	var state securityGroupResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewUpdateSecurityGroupOpts(plan.Name.ValueString())

	sdkSG, httpResponse, err := s.PubliccloudAPI.
		UpdateSecurityGroup(ctx, state.ID.ValueString()).
		UpdateSecurityGroupOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptSecurityGroupToResource(*sdkSG)
	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (s *securityGroupResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state securityGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	httpResponse, err := s.PubliccloudAPI.DeleteSecurityGroup(
		ctx,
		state.ID.ValueString(),
	).Execute()

	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
	}
}

func NewSecurityGroupResource() resource.Resource {
	return &securityGroupResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_security_group",
		},
	}
}
