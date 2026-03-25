package publiccloud

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &targetGroupTargetsResource{}
	_ resource.ResourceWithImportState = &targetGroupTargetsResource{}
)

type targetGroupTargetsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TargetGroupID types.String `tfsdk:"target_group_id"`
	InstanceIDs   types.Set    `tfsdk:"instance_ids"`
}

type targetGroupTargetsResource struct {
	utils.ResourceAPI
}

func (t *targetGroupTargetsResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	// Import by target_group_id
	state := targetGroupTargetsResourceModel{
		ID:            types.StringValue(request.ID),
		TargetGroupID: types.StringValue(request.ID),
	}

	// Read current targets from API
	sdkResult, httpResponse, err := t.PubliccloudAPI.
		GetTargetList(ctx, request.ID).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	instanceIDs := make([]string, 0)
	for _, target := range sdkResult.GetTargets() {
		instanceIDs = append(instanceIDs, target.GetId())
	}

	ids, diags := types.SetValueFrom(ctx, types.StringType, instanceIDs)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.InstanceIDs = ids

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (t *targetGroupTargetsResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	response *resource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: utils.BetaDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The target group ID (same as target_group_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the target group to register instances in",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_ids": schema.SetAttribute{
				Required:    true,
				Description: "Set of instance IDs to register as targets in the target group",
				ElementType: types.StringType,
			},
		},
	}
}

func (t *targetGroupTargetsResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan targetGroupTargetsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	var instanceIDs []string
	response.Diagnostics.Append(plan.InstanceIDs.ElementsAs(ctx, &instanceIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(instanceIDs) > 0 {
		httpResponse, err := t.PubliccloudAPI.
			RegisterTargets(ctx, plan.TargetGroupID.ValueString()).
			RequestBody(instanceIDs).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.TargetGroupID

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (t *targetGroupTargetsResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state targetGroupTargetsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkResult, httpResponse, err := t.PubliccloudAPI.
		GetTargetList(ctx, state.TargetGroupID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	// Get the list of instance IDs we manage from state
	var managedIDs []string
	response.Diagnostics.Append(state.InstanceIDs.ElementsAs(ctx, &managedIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}
	managedSet := make(map[string]bool, len(managedIDs))
	for _, id := range managedIDs {
		managedSet[id] = true
	}

	// Filter: only keep targets that we manage
	currentIDs := make([]string, 0)
	for _, target := range sdkResult.GetTargets() {
		if managedSet[target.GetId()] {
			currentIDs = append(currentIDs, target.GetId())
		}
	}

	ids, diags := types.SetValueFrom(ctx, types.StringType, currentIDs)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.InstanceIDs = ids

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (t *targetGroupTargetsResource) Update(
	ctx context.Context,
	request resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	var plan targetGroupTargetsResourceModel
	var state targetGroupTargetsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var planIDs []string
	var stateIDs []string
	response.Diagnostics.Append(plan.InstanceIDs.ElementsAs(ctx, &planIDs, false)...)
	response.Diagnostics.Append(state.InstanceIDs.ElementsAs(ctx, &stateIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	planSet := make(map[string]bool, len(planIDs))
	for _, id := range planIDs {
		planSet[id] = true
	}
	stateSet := make(map[string]bool, len(stateIDs))
	for _, id := range stateIDs {
		stateSet[id] = true
	}

	// Compute additions and removals
	toAdd := make([]string, 0)
	for _, id := range planIDs {
		if !stateSet[id] {
			toAdd = append(toAdd, id)
		}
	}
	toRemove := make([]string, 0)
	for _, id := range stateIDs {
		if !planSet[id] {
			toRemove = append(toRemove, id)
		}
	}

	// Sort for determinism
	sort.Strings(toAdd)
	sort.Strings(toRemove)

	targetGroupID := plan.TargetGroupID.ValueString()

	if len(toRemove) > 0 {
		httpResponse, err := t.PubliccloudAPI.
			DeregisterTargets(ctx, targetGroupID).
			RequestBody(toRemove).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	if len(toAdd) > 0 {
		httpResponse, err := t.PubliccloudAPI.
			RegisterTargets(ctx, targetGroupID).
			RequestBody(toAdd).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.TargetGroupID

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (t *targetGroupTargetsResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state targetGroupTargetsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var instanceIDs []string
	response.Diagnostics.Append(state.InstanceIDs.ElementsAs(ctx, &instanceIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(instanceIDs) > 0 {
		httpResponse, err := t.PubliccloudAPI.
			DeregisterTargets(ctx, state.TargetGroupID.ValueString()).
			RequestBody(instanceIDs).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		}
	}
}

func NewTargetGroupTargetsResource() resource.Resource {
	return &targetGroupTargetsResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_target_group_targets",
		},
	}
}

