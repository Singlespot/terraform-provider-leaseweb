package publiccloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leaseweb/leaseweb-go-sdk/publiccloud"
	"github.com/leaseweb/terraform-provider-leaseweb/internal/utils"
)

var (
	_ resource.ResourceWithConfigure   = &securityGroupRulesResource{}
	_ resource.ResourceWithImportState = &securityGroupRulesResource{}
)

type firewallRuleModel struct {
	Protocol  types.String `tfsdk:"protocol"`
	StartPort types.Int32  `tfsdk:"start_port"`
	EndPort   types.Int32  `tfsdk:"end_port"`
	IcmpType  types.Int32  `tfsdk:"icmp_type"`
	IcmpCode  types.Int32  `tfsdk:"icmp_code"`
	Source    types.String `tfsdk:"source"`
}

type securityGroupRulesResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SecurityGroupID types.String `tfsdk:"security_group_id"`
	Rules           types.Set    `tfsdk:"rules"`
}

type securityGroupRulesResource struct {
	utils.ResourceAPI
}

func (r *securityGroupRulesResource) ImportState(
	ctx context.Context,
	request resource.ImportStateRequest,
	response *resource.ImportStateResponse,
) {
	// Import by security_group_id
	state := securityGroupRulesResourceModel{
		ID:              types.StringValue(request.ID),
		SecurityGroupID: types.StringValue(request.ID),
	}

	// Read current rules from API and populate state
	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetSecurityGroupFirewallRules(ctx, request.ID).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	rules := make([]firewallRuleModel, 0)
	for _, sdkRule := range sdkResult.GetFirewallRules() {
		rule := firewallRuleModel{
			Protocol: types.StringValue(sdkRule.GetProtocol()),
			Source:   types.StringValue(sdkRule.GetSource()),
		}
		if sp, ok := sdkRule.GetStartPortOk(); ok && sp != nil {
			rule.StartPort = types.Int32Value(*sp)
		} else {
			rule.StartPort = types.Int32Null()
		}
		if ep, ok := sdkRule.GetEndPortOk(); ok && ep != nil {
			rule.EndPort = types.Int32Value(*ep)
		} else {
			rule.EndPort = types.Int32Null()
		}
		if it, ok := sdkRule.GetIcmpTypeOk(); ok && it != nil {
			rule.IcmpType = types.Int32Value(*it)
		} else {
			rule.IcmpType = types.Int32Null()
		}
		if ic, ok := sdkRule.GetIcmpCodeOk(); ok && ic != nil {
			rule.IcmpCode = types.Int32Value(*ic)
		} else {
			rule.IcmpCode = types.Int32Null()
		}
		rules = append(rules, rule)
	}

	rulesSet, diags := types.SetValueFrom(ctx, firewallRuleObjectType(), rules)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.Rules = rulesSet

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *securityGroupRulesResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	response *resource.SchemaResponse,
) {
	response.Schema = schema.Schema{
		Description: utils.BetaDescription + " Manages the complete set of firewall rules for a security group. All rules are replaced on each apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the security group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rules": schema.SetNestedAttribute{
				Required:    true,
				Description: "Set of firewall rules to authorize",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"protocol": schema.StringAttribute{
							Required:    true,
							Description: "Protocol: TCP, UDP, or ICMP",
							Validators: []validator.String{
								stringvalidator.OneOf("TCP", "UDP", "ICMP"),
							},
						},
						"start_port": schema.Int32Attribute{
							Optional:    true,
							Description: "Start port (for TCP/UDP)",
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
							},
						},
						"end_port": schema.Int32Attribute{
							Optional:    true,
							Description: "End port (for TCP/UDP)",
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
							},
						},
						"icmp_type": schema.Int32Attribute{
							Optional:    true,
							Description: "ICMP type (for ICMP)",
						},
						"icmp_code": schema.Int32Attribute{
							Optional:    true,
							Description: "ICMP code (for ICMP)",
						},
						"source": schema.StringAttribute{
							Optional:    true,
							Description: "Source IP address or CIDR block",
						},
					},
				},
			},
		},
	}
}

func (r *securityGroupRulesResource) Create(
	ctx context.Context,
	request resource.CreateRequest,
	response *resource.CreateResponse,
) {
	var plan securityGroupRulesResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	var rules []firewallRuleModel
	response.Diagnostics.Append(plan.Rules.ElementsAs(ctx, &rules, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(rules) > 0 {
		sdkRules := adaptFirewallRulesToSDK(rules)
		opts := publiccloud.NewAuthorizeFirewallRulesOpts(sdkRules)

		httpResponse, err := r.PubliccloudAPI.
			AuthorizeSecurityGroupFirewallRules(ctx, plan.SecurityGroupID.ValueString()).
			AuthorizeFirewallRulesOpts(*opts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.SecurityGroupID
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *securityGroupRulesResource) Read(
	ctx context.Context,
	request resource.ReadRequest,
	response *resource.ReadResponse,
) {
	var state securityGroupRulesResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetSecurityGroupFirewallRules(ctx, state.SecurityGroupID.ValueString()).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	rules := make([]firewallRuleModel, 0)
	for _, sdkRule := range sdkResult.GetFirewallRules() {
		rule := firewallRuleModel{
			Protocol: types.StringValue(sdkRule.GetProtocol()),
		}
		src := sdkRule.GetSource()
		if src != "" {
			rule.Source = types.StringValue(src)
		} else {
			rule.Source = types.StringNull()
		}
		if sp, ok := sdkRule.GetStartPortOk(); ok && sp != nil {
			rule.StartPort = types.Int32Value(*sp)
		} else {
			rule.StartPort = types.Int32Null()
		}
		if ep, ok := sdkRule.GetEndPortOk(); ok && ep != nil {
			rule.EndPort = types.Int32Value(*ep)
		} else {
			rule.EndPort = types.Int32Null()
		}
		if it, ok := sdkRule.GetIcmpTypeOk(); ok && it != nil {
			rule.IcmpType = types.Int32Value(*it)
		} else {
			rule.IcmpType = types.Int32Null()
		}
		if ic, ok := sdkRule.GetIcmpCodeOk(); ok && ic != nil {
			rule.IcmpCode = types.Int32Value(*ic)
		} else {
			rule.IcmpCode = types.Int32Null()
		}
		rules = append(rules, rule)
	}

	rulesSet, diags := types.SetValueFrom(ctx, firewallRuleObjectType(), rules)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	state.Rules = rulesSet

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *securityGroupRulesResource) Update(
	ctx context.Context,
	request resource.UpdateRequest,
	response *resource.UpdateResponse,
) {
	var plan securityGroupRulesResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	sgID := plan.SecurityGroupID.ValueString()

	// Step 1: get current rule IDs and revoke them all
	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetSecurityGroupFirewallRules(ctx, sgID).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	// Collect existing rule identifiers for revocation
	// The revoke API expects rule identifiers; we build them from the response
	existingRules := sdkResult.GetFirewallRules()
	if len(existingRules) > 0 {
		ruleIDs := make([]string, 0, len(existingRules))
		for _, rule := range existingRules {
			ruleIDs = append(ruleIDs, buildRuleIdentifier(rule))
		}
		revokeOpts := publiccloud.NewRevokeFirewallRulesOpts(ruleIDs)
		httpResponse, err := r.PubliccloudAPI.
			RevokeSecurityGroupFirewallRules(ctx, sgID).
			RevokeFirewallRulesOpts(*revokeOpts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	// Step 2: authorize new rules
	var rules []firewallRuleModel
	response.Diagnostics.Append(plan.Rules.ElementsAs(ctx, &rules, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(rules) > 0 {
		sdkRules := adaptFirewallRulesToSDK(rules)
		authorizeOpts := publiccloud.NewAuthorizeFirewallRulesOpts(sdkRules)

		httpResponse, err := r.PubliccloudAPI.
			AuthorizeSecurityGroupFirewallRules(ctx, sgID).
			AuthorizeFirewallRulesOpts(*authorizeOpts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
			return
		}
	}

	plan.ID = plan.SecurityGroupID
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *securityGroupRulesResource) Delete(
	ctx context.Context,
	request resource.DeleteRequest,
	response *resource.DeleteResponse,
) {
	var state securityGroupRulesResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	sgID := state.SecurityGroupID.ValueString()

	sdkResult, httpResponse, err := r.PubliccloudAPI.
		GetSecurityGroupFirewallRules(ctx, sgID).
		Execute()
	if err != nil {
		utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		return
	}

	existingRules := sdkResult.GetFirewallRules()
	if len(existingRules) > 0 {
		ruleIDs := make([]string, 0, len(existingRules))
		for _, rule := range existingRules {
			ruleIDs = append(ruleIDs, buildRuleIdentifier(rule))
		}
		revokeOpts := publiccloud.NewRevokeFirewallRulesOpts(ruleIDs)
		httpResponse, err := r.PubliccloudAPI.
			RevokeSecurityGroupFirewallRules(ctx, sgID).
			RevokeFirewallRulesOpts(*revokeOpts).
			Execute()
		if err != nil {
			utils.SdkError(ctx, &response.Diagnostics, err, httpResponse)
		}
	}
}

func adaptFirewallRulesToSDK(rules []firewallRuleModel) []publiccloud.AuthorizeRule {
	sdkRules := make([]publiccloud.AuthorizeRule, 0, len(rules))
	for _, rule := range rules {
		sdkRule := publiccloud.NewAuthorizeRule(rule.Protocol.ValueString())
		if !rule.StartPort.IsNull() {
			sdkRule.SetStartPort(rule.StartPort.ValueInt32())
		}
		if !rule.EndPort.IsNull() {
			sdkRule.SetEndPort(rule.EndPort.ValueInt32())
		}
		if !rule.IcmpType.IsNull() {
			sdkRule.SetIcmpType(rule.IcmpType.ValueInt32())
		}
		if !rule.IcmpCode.IsNull() {
			sdkRule.SetIcmpCode(rule.IcmpCode.ValueInt32())
		}
		if !rule.Source.IsNull() {
			sdkRule.SetSource(rule.Source.ValueString())
		}
		sdkRules = append(sdkRules, *sdkRule)
	}
	return sdkRules
}

// buildRuleIdentifier creates a deterministic string identifier for a firewall rule
// to be used with the revoke API. The revoke API takes rule IDs which are
// composite keys of the rule attributes.
func buildRuleIdentifier(rule publiccloud.SecurityGroupFirewallRuleResponse) string {
	parts := []string{rule.GetProtocol()}
	if sp, ok := rule.GetStartPortOk(); ok && sp != nil {
		parts = append(parts, fmt.Sprintf("%d", *sp))
	}
	if ep, ok := rule.GetEndPortOk(); ok && ep != nil {
		parts = append(parts, fmt.Sprintf("%d", *ep))
	}
	parts = append(parts, rule.GetSource())
	return strings.Join(parts, ":")
}

func firewallRuleObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"protocol":   types.StringType,
			"start_port": types.Int32Type,
			"end_port":   types.Int32Type,
			"icmp_type":  types.Int32Type,
			"icmp_code":  types.Int32Type,
			"source":     types.StringType,
		},
	}
}

func NewSecurityGroupRulesResource() resource.Resource {
	return &securityGroupRulesResource{
		ResourceAPI: utils.ResourceAPI{
			Name: "public_cloud_security_group_rules",
		},
	}
}
