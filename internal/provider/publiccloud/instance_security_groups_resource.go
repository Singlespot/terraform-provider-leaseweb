package publiccloud

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leaseweb/leaseweb-go-sdk/publiccloud"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &instanceSecurityGroupsResource{}
	_ resource.ResourceWithImportState = &instanceSecurityGroupsResource{}
)

type instanceSecurityGroupsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	InstanceID       types.String `tfsdk:"instance_id"`
	SecurityGroupIDs types.Set    `tfsdk:"security_group_ids"`
}

type instanceSecurityGroupsResource struct {
	utils.ResourceAPI
}

func (r *instanceSecurityGroupsResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	// Import by instance_id
	state := instanceSecurityGroupsResourceModel{
		ID:         types.StringValue(request.ID),
		InstanceID: types.StringValue(request.ID),
	}

	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetInstanceSecurityGroups(ctx, request.ID).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	sgIDs := make([]string, 0)
	for _, sg := range sdkResult.GetSecurityGroups() {
		sgIDs = append(sgIDs, sg.GetId())
	}

	ids, diags := types.SetValueFrom(ctx, types.StringType, sgIDs)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.SecurityGroupIDs = ids

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *instanceSecurityGroupsResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	response *resource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: utils.BetaDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The instance ID (same as instance_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the instance to attach security groups to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group_ids": schema.SetAttribute{
				Required:    true,
				Description: "Set of security group IDs to attach to the instance",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *instanceSecurityGroupsResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan instanceSecurityGroupsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	var sgIDs []string
	response.Diagnostics.Append(plan.SecurityGroupIDs.ElementsAs(ctx, &sgIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(sgIDs) > 0 {
		opts := publiccloud.NewAttachSecurityGroupsOpts(sgIDs)
		httpResponse, err := r.PubliccloudAPI.
			AttachSecurityGroups(ctx, plan.InstanceID.ValueString()).
			AttachSecurityGroupsOpts(*opts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.InstanceID
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *instanceSecurityGroupsResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state instanceSecurityGroupsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetInstanceSecurityGroups(ctx, state.InstanceID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	// Get the list of SG IDs we manage from state
	var managedIDs []string
	response.Diagnostics.Append(state.SecurityGroupIDs.ElementsAs(ctx, &managedIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}
	managedSet := make(map[string]bool, len(managedIDs))
	for _, id := range managedIDs {
		managedSet[id] = true
	}

	// Filter: only keep SGs that we manage
	currentIDs := make([]string, 0)
	for _, sg := range sdkResult.GetSecurityGroups() {
		if managedSet[sg.GetId()] {
			currentIDs = append(currentIDs, sg.GetId())
		}
	}

	ids, diags := types.SetValueFrom(ctx, types.StringType, currentIDs)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.SecurityGroupIDs = ids

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *instanceSecurityGroupsResource) Update(
	ctx context.Context,
	request resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	var plan instanceSecurityGroupsResourceModel
	var state instanceSecurityGroupsResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var planIDs []string
	var stateIDs []string
	response.Diagnostics.Append(plan.SecurityGroupIDs.ElementsAs(ctx, &planIDs, false)...)
	response.Diagnostics.Append(state.SecurityGroupIDs.ElementsAs(ctx, &stateIDs, false)...)
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

	sort.Strings(toAdd)
	sort.Strings(toRemove)

	instanceID := plan.InstanceID.ValueString()

	if len(toRemove) > 0 {
		opts := publiccloud.NewDetachSecurityGroupsOpts(toRemove)
		httpResponse, err := r.PubliccloudAPI.
			DetachSecurityGroups(ctx, instanceID).
			DetachSecurityGroupsOpts(*opts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	if len(toAdd) > 0 {
		opts := publiccloud.NewAttachSecurityGroupsOpts(toAdd)
		httpResponse, err := r.PubliccloudAPI.
			AttachSecurityGroups(ctx, instanceID).
			AttachSecurityGroupsOpts(*opts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.InstanceID
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *instanceSecurityGroupsResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state instanceSecurityGroupsResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var sgIDs []string
	response.Diagnostics.Append(state.SecurityGroupIDs.ElementsAs(ctx, &sgIDs, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(sgIDs) > 0 {
		opts := publiccloud.NewDetachSecurityGroupsOpts(sgIDs)
		httpResponse, err := r.PubliccloudAPI.
			DetachSecurityGroups(ctx, state.InstanceID.ValueString()).
			DetachSecurityGroupsOpts(*opts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		}
	}
}

func NewInstanceSecurityGroupsResource() resource.Resource {
	return &instanceSecurityGroupsResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_instance_security_groups",
		},
	}
}
