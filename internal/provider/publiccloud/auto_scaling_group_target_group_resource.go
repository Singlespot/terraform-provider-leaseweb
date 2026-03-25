package publiccloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leaseweb/leaseweb-go-sdk/publiccloud"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &autoScalingGroupTargetGroupResource{}
	_ resource.ResourceWithImportState = &autoScalingGroupTargetGroupResource{}
)

type autoScalingGroupTargetGroupResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	AutoScalingGroupID types.String `tfsdk:"auto_scaling_group_id"`
	TargetGroupID      types.String `tfsdk:"target_group_id"`
}

type autoScalingGroupTargetGroupResource struct {
	utils.ResourceAPI
}

func (r *autoScalingGroupTargetGroupResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	// Import format: auto_scaling_group_id/target_group_id
	parts := strings.SplitN(request.ID, "/", 2)
	if len(parts) != 2 {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: auto_scaling_group_id/target_group_id, got: %s", request.ID),
		)
		return
	}

	state := autoScalingGroupTargetGroupResourceModel{
		ID:                 types.StringValue(request.ID),
		AutoScalingGroupID: types.StringValue(parts[0]),
		TargetGroupID:      types.StringValue(parts[1]),
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *autoScalingGroupTargetGroupResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	response *resource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: utils.BetaDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite ID (auto_scaling_group_id/target_group_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_scaling_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the auto scaling group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the target group to associate",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *autoScalingGroupTargetGroupResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan autoScalingGroupTargetGroupResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewTargetGroupIdOpts(plan.TargetGroupID.ValueString())

	_, httpResponse, err := r.PubliccloudAPI.
		RegisterAutoScalingGroupTargetGroup(ctx, plan.AutoScalingGroupID.ValueString()).
		TargetGroupIdOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	plan.ID = types.StringValue(
		plan.AutoScalingGroupID.ValueString() + "/" + plan.TargetGroupID.ValueString(),
	)

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *autoScalingGroupTargetGroupResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state autoScalingGroupTargetGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkASG, httpResponse, err := r.PubliccloudAPI.
		GetAutoScalingGroup(ctx, state.AutoScalingGroupID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	// Check if target group is still associated
	found := false
	for _, tg := range sdkASG.GetTargetGroups() {
		if tg.GetId() == state.TargetGroupID.ValueString() {
			found = true
			break
		}
	}

	if !found {
		response.State.RemoveResource(ctx)
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *autoScalingGroupTargetGroupResource) Update(
	_ context.Context,
	_ resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	// Both fields require replacement, so Update should never be called
	response.Diagnostics.AddError(
		"Update not supported",
		"This resource does not support in-place updates. Changes require replacement.",
	)
}

func (r *autoScalingGroupTargetGroupResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state autoScalingGroupTargetGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewTargetGroupIdOpts(state.TargetGroupID.ValueString())

	_, httpResponse, err := r.PubliccloudAPI.
		DeregisterAutoScalingGroupTargetGroup(ctx, state.AutoScalingGroupID.ValueString()).
		TargetGroupIdOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
	}
}

func NewAutoScalingGroupTargetGroupResource() resource.Resource {
	return &autoScalingGroupTargetGroupResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_auto_scaling_group_target_group",
		},
	}
}
