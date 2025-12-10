package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client *Client
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Metadata      types.Map    `tfsdk:"metadata"`
	RetentionDays types.Int64  `tfsdk:"retention_days"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Langfuse project resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name",
				Required:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Project metadata",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"retention_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days to retain data. Set to 0 for no retention limit.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project last update timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	createReq := CreateProjectRequest{
		Name: data.Name.ValueString(),
	}

	// Handle metadata
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		metadataMap := make(map[string]interface{})
		for key, value := range data.Metadata.Elements() {
			if strValue, ok := value.(types.String); ok {
				metadataMap[key] = strValue.ValueString()
			}
		}
		createReq.Metadata = metadataMap
	}

	// Handle retention days
	if !data.RetentionDays.IsNull() && !data.RetentionDays.IsUnknown() {
		retention := int(data.RetentionDays.ValueInt64())
		createReq.Retention = &retention
	}

	// Create project
	project, err := r.client.CreateProject(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}

	// Store the project ID first
	data.ID = types.StringValue(project.ID)

	// Read the project back to ensure we have all computed fields properly set
	// This fixes issues where the API create response doesn't include all fields
	readProject, err := r.client.GetProject(project.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project after creation, got error: %s", err))
		return
	}

	// Update model with complete data from read operation
	data.Name = types.StringValue(readProject.Name)
	data.CreatedAt = types.StringValue(readProject.CreatedAt)
	data.UpdatedAt = types.StringValue(readProject.UpdatedAt)

	if readProject.RetentionDays != nil {
		data.RetentionDays = types.Int64Value(int64(*readProject.RetentionDays))
	}

	// Handle metadata response
	if readProject.Metadata != nil && len(readProject.Metadata) > 0 {
		metadataElements := make(map[string]types.String)
		for key, value := range readProject.Metadata {
			if strValue, ok := value.(string); ok {
				metadataElements[key] = types.StringValue(strValue)
			}
		}
		metadataValueMap := make(map[string]attr.Value)
		for key, value := range metadataElements {
			metadataValueMap[key] = value
		}
		metadataMap, diags := types.MapValue(types.StringType, metadataValueMap)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Metadata = metadataMap
		}
	}

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get project from API
	project, err := r.client.GetProject(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	// Update model with fresh data
	data.ID = types.StringValue(project.ID)
	data.Name = types.StringValue(project.Name)
	data.CreatedAt = types.StringValue(project.CreatedAt)
	data.UpdatedAt = types.StringValue(project.UpdatedAt)

	if project.RetentionDays != nil {
		data.RetentionDays = types.Int64Value(int64(*project.RetentionDays))
	}

	// Handle metadata response
	if project.Metadata != nil && len(project.Metadata) > 0 {
		metadataValueMap := make(map[string]attr.Value)
		for key, value := range project.Metadata {
			if strValue, ok := value.(string); ok {
				metadataValueMap[key] = types.StringValue(strValue)
			}
		}
		metadataMap, diags := types.MapValue(types.StringType, metadataValueMap)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Metadata = metadataMap
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	updateReq := UpdateProjectRequest{
		Name: data.Name.ValueString(),
	}

	// Handle metadata
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		metadataMap := make(map[string]interface{})
		for key, value := range data.Metadata.Elements() {
			if strValue, ok := value.(types.String); ok {
				metadataMap[key] = strValue.ValueString()
			}
		}
		updateReq.Metadata = metadataMap
	}

	// Handle retention days
	if !data.RetentionDays.IsNull() && !data.RetentionDays.IsUnknown() {
		retention := int(data.RetentionDays.ValueInt64())
		updateReq.Retention = &retention
	}

	// Update project
	project, err := r.client.UpdateProject(data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	// Update model with response data
	data.Name = types.StringValue(project.Name)
	data.CreatedAt = types.StringValue(project.CreatedAt)
	data.UpdatedAt = types.StringValue(project.UpdatedAt)

	if project.RetentionDays != nil {
		data.RetentionDays = types.Int64Value(int64(*project.RetentionDays))
	}

	// Handle metadata response
	if project.Metadata != nil && len(project.Metadata) > 0 {
		metadataValueMap := make(map[string]attr.Value)
		for key, value := range project.Metadata {
			if strValue, ok := value.(string); ok {
				metadataValueMap[key] = types.StringValue(strValue)
			}
		}
		metadataMap, diags := types.MapValue(types.StringType, metadataValueMap)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Metadata = metadataMap
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete project
	err := r.client.DeleteProject(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
