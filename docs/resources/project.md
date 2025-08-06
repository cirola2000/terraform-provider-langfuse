# langfuse_project Resource

Manages a Langfuse project. Projects in Langfuse are used to organize and separate different applications or environments.

## Example Usage

### Basic Usage

```hcl
resource "langfuse_project" "example" {
  name = "my-application"
}
```

### Complete Usage

```hcl
resource "langfuse_project" "example" {
  name = "my-application"
  
  metadata = {
    environment   = "production"
    team          = "data-team"
    cost_center   = "engineering"
    application   = "web-app"
  }
  
  retention_days = 90
}
```

### With No Retention Limit

```hcl
resource "langfuse_project" "unlimited" {
  name = "unlimited-retention-project"
  
  metadata = {
    environment = "production"
    type        = "long-term-storage"
  }
  
  retention_days = 0  # No retention limit
}
```

### With Variable Configuration

```hcl
variable "project_config" {
  description = "Project configuration"
  type = object({
    name           = string
    environment    = string
    retention_days = number
  })
  default = {
    name           = "my-project"
    environment    = "production"
    retention_days = 0  # No retention limit
  }
}

resource "langfuse_project" "example" {
  name = var.project_config.name
  
  metadata = {
    environment = var.project_config.environment
    managed_by  = "terraform"
  }
  
  retention_days = var.project_config.retention_days
}
```

## Schema

### Required

- `name` (String) The name of the project. Must be unique within your Langfuse organization.

### Optional

- `metadata` (Map of String) Key-value pairs to store additional information about the project. Useful for tagging, categorization, and organization.
- `retention_days` (Number) Number of days to retain project data. Set to 0 for no retention limit. Defaults to 0 (no retention limit).

### Read-Only

- `id` (String) The unique identifier of the project.
- `created_at` (String) The timestamp when the project was created (RFC3339 format).
- `updated_at` (String) The timestamp when the project was last updated (RFC3339 format).

## Import

Projects can be imported using their ID:

```bash
terraform import langfuse_project.example project-id-here
```

To find the project ID, you can:
1. Check the Langfuse web interface
2. Use the Langfuse API to list projects
3. Check the Terraform state after creation 