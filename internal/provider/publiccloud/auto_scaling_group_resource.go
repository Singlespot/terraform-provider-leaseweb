package publiccloud

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/leaseweb/leaseweb-go-sdk/publiccloud"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &autoScalingGroupResource{}
	_ resource.ResourceWithImportState = &autoScalingGroupResource{}
)

type autoScalingGroupResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Type          types.String `tfsdk:"type"`
	State         types.String `tfsdk:"state"`
	DesiredAmount types.Int32  `tfsdk:"desired_amount"`
	MinimumAmount types.Int32  `tfsdk:"minimum_amount"`
	MaximumAmount types.Int32  `tfsdk:"maximum_amount"`
	CpuThreshold  types.Int32  `tfsdk:"cpu_threshold"`
	WarmupTime    types.Int32  `tfsdk:"warmup_time"`
	CooldownTime  types.Int32  `tfsdk:"cooldown_time"`
	Region        types.String `tfsdk:"region"`
	Reference     types.String `tfsdk:"reference"`
	InstanceID    types.String `tfsdk:"instance_id"`
	StartsAt      types.String `tfsdk:"starts_at"`
	EndsAt        types.String `tfsdk:"ends_at"`
}

func adaptAutoScalingGroupDetailsToResource(
	sdkASG publiccloud.AutoScalingGroupDetails,
) *autoScalingGroupResourceModel {
	model := autoScalingGroupResourceModel{
		ID:        basetypes.NewStringValue(sdkASG.GetId()),
		Type:      basetypes.NewStringValue(string(sdkASG.GetType())),
		State:     basetypes.NewStringValue(string(sdkASG.GetState())),
		Region:    basetypes.NewStringValue(string(sdkASG.GetRegion())),
		Reference: basetypes.NewStringValue(sdkASG.GetReference()),
	}

	if desiredAmount, ok := sdkASG.GetDesiredAmountOk(); ok && desiredAmount != nil {
		model.DesiredAmount = basetypes.NewInt32Value(*desiredAmount)
	} else {
		model.DesiredAmount = basetypes.NewInt32Null()
	}

	if minimumAmount, ok := sdkASG.GetMinimumAmountOk(); ok && minimumAmount != nil {
		model.MinimumAmount = basetypes.NewInt32Value(*minimumAmount)
	} else {
		model.MinimumAmount = basetypes.NewInt32Null()
	}

	if maximumAmount, ok := sdkASG.GetMaximumAmountOk(); ok && maximumAmount != nil {
		model.MaximumAmount = basetypes.NewInt32Value(*maximumAmount)
	} else {
		model.MaximumAmount = basetypes.NewInt32Null()
	}

	if cpuThreshold, ok := sdkASG.GetCpuThresholdOk(); ok && cpuThreshold != nil {
		model.CpuThreshold = basetypes.NewInt32Value(*cpuThreshold)
	} else {
		model.CpuThreshold = basetypes.NewInt32Null()
	}

	if warmupTime, ok := sdkASG.GetWarmupTimeOk(); ok && warmupTime != nil {
		model.WarmupTime = basetypes.NewInt32Value(*warmupTime)
	} else {
		model.WarmupTime = basetypes.NewInt32Null()
	}

	if cooldownTime, ok := sdkASG.GetCooldownTimeOk(); ok && cooldownTime != nil {
		model.CooldownTime = basetypes.NewInt32Value(*cooldownTime)
	} else {
		model.CooldownTime = basetypes.NewInt32Null()
	}

	if startsAt, ok := sdkASG.GetStartsAtOk(); ok && startsAt != nil {
		model.StartsAt = basetypes.NewStringValue(startsAt.String())
	} else {
		model.StartsAt = basetypes.NewStringNull()
	}

	if endsAt, ok := sdkASG.GetEndsAtOk(); ok && endsAt != nil {
		model.EndsAt = basetypes.NewStringValue(endsAt.String())
	} else {
		model.EndsAt = basetypes.NewStringNull()
	}

	return &model
}

type autoScalingGroupResource struct {
	utils.ResourceAPI
}

func (a *autoScalingGroupResource) ImportState(
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

func (a *autoScalingGroupResource) Schema(
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
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Auto scaling group type. Valid options are `MANUAL`, `SCHEDULED`, `CPU_BASED`",
				Validators: []validator.String{
					stringvalidator.OneOf(utils.AdaptStringTypeArrayToStringArray(publiccloud.AllowedAutoScalingGroupTypeEnumValues)...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the auto scaling group",
			},
			"desired_amount": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for MANUAL and SCHEDULED. Number of instances to be launched",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"minimum_amount": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for CPU_BASED. The minimum number of instances that should be running",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"maximum_amount": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for CPU_BASED. The maximum number of instances that can be running",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"cpu_threshold": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for CPU_BASED. The target average CPU utilization for scaling (1-100)",
				Validators: []validator.Int32{
					int32validator.Between(1, 100),
				},
			},
			"warmup_time": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for CPU_BASED. Warm-up time in seconds for new instances",
				Validators: []validator.Int32{
					int32validator.AtLeast(0),
				},
			},
			"cooldown_time": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for CPU_BASED. Cool-down time in seconds for new instances",
				Validators: []validator.Int32{
					int32validator.AtLeast(0),
				},
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The region of the auto scaling group (inherited from source instance)",
			},
			"reference": schema.StringAttribute{
				Required:    true,
				Description: "The identifying name set to the auto scaling group",
			},
			"instance_id": schema.StringAttribute{
				Required:    true,
				Description: "The instance on which new instances will be based. Must be Running or Stopped",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"starts_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for SCHEDULED. Date and time (UTC) that the instances need to be launched",
			},
			"ends_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Required for SCHEDULED. Date and time (UTC) that the instances need to be terminated",
			},
		},
	}
}

func (a *autoScalingGroupResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan autoScalingGroupResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewCreateAutoScalingGroupOpts(
		plan.InstanceID.ValueString(),
		plan.Reference.ValueString(),
		plan.Type.ValueString(),
	)

	if !plan.DesiredAmount.IsNull() && !plan.DesiredAmount.IsUnknown() {
		opts.SetDesiredAmount(plan.DesiredAmount.ValueInt32())
	}
	if !plan.MinimumAmount.IsNull() && !plan.MinimumAmount.IsUnknown() {
		opts.SetMinimumAmount(plan.MinimumAmount.ValueInt32())
	}
	if !plan.MaximumAmount.IsNull() && !plan.MaximumAmount.IsUnknown() {
		opts.SetMaximumAmount(plan.MaximumAmount.ValueInt32())
	}
	if !plan.CpuThreshold.IsNull() && !plan.CpuThreshold.IsUnknown() {
		opts.SetCpuThreshold(plan.CpuThreshold.ValueInt32())
	}
	if !plan.WarmupTime.IsNull() && !plan.WarmupTime.IsUnknown() {
		opts.SetWarmupTime(plan.WarmupTime.ValueInt32())
	}
	if !plan.CooldownTime.IsNull() && !plan.CooldownTime.IsUnknown() {
		opts.SetCooldownTime(plan.CooldownTime.ValueInt32())
	}

	sdkASG, httpResponse, err := a.PubliccloudAPI.CreateAutoScalingGroup(ctx).
		CreateAutoScalingGroupOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptAutoScalingGroupDetailsToResource(*sdkASG)
	model.InstanceID = plan.InstanceID

	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (a *autoScalingGroupResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state autoScalingGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkASG, httpResponse, err := a.PubliccloudAPI.
		GetAutoScalingGroup(ctx, state.ID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptAutoScalingGroupDetailsToResource(*sdkASG)
	// Preserve instance_id from state (not returned by Get)
	model.InstanceID = state.InstanceID

	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (a *autoScalingGroupResource) Update(
	ctx context.Context,
	request resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	var plan autoScalingGroupResourceModel
	var state autoScalingGroupResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	opts := publiccloud.NewUpdateAutoScalingGroupOpts()
	opts.SetReference(plan.Reference.ValueString())

	if !plan.DesiredAmount.IsNull() && !plan.DesiredAmount.IsUnknown() {
		opts.SetDesiredAmount(plan.DesiredAmount.ValueInt32())
	}
	if !plan.MinimumAmount.IsNull() && !plan.MinimumAmount.IsUnknown() {
		opts.SetMinimumAmount(plan.MinimumAmount.ValueInt32())
	}
	if !plan.MaximumAmount.IsNull() && !plan.MaximumAmount.IsUnknown() {
		opts.SetMaximumAmount(plan.MaximumAmount.ValueInt32())
	}
	if !plan.CpuThreshold.IsNull() && !plan.CpuThreshold.IsUnknown() {
		opts.SetCpuThreshold(plan.CpuThreshold.ValueInt32())
	}
	if !plan.WarmupTime.IsNull() && !plan.WarmupTime.IsUnknown() {
		opts.SetWarmupTime(plan.WarmupTime.ValueInt32())
	}
	if !plan.CooldownTime.IsNull() && !plan.CooldownTime.IsUnknown() {
		opts.SetCooldownTime(plan.CooldownTime.ValueInt32())
	}

	sdkASG, httpResponse, err := a.PubliccloudAPI.
		UpdateAutoScalingGroup(ctx, state.ID.ValueString()).
		UpdateAutoScalingGroupOpts(*opts).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	model := adaptAutoScalingGroupDetailsToResource(*sdkASG)
	model.InstanceID = state.InstanceID

	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (a *autoScalingGroupResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state autoScalingGroupResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	httpResponse, err := a.PubliccloudAPI.DeleteAutoScalingGroup(
		ctx,
		state.ID.ValueString(),
	).Execute()

	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
	}
}

func NewAutoScalingGroupResource() resource.Resource {
	return &autoScalingGroupResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_auto_scaling_group",
		},
	}
}
