variable "project_name" {
  description = "Project name used when naming Azure resources."
  type        = string
  default     = "rtsp-stream-console"
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "demo"
}

variable "location" {
  description = "Azure region for the deployment."
  type        = string
  default     = "centralindia"
}

variable "vm_size" {
  description = "Azure VM SKU used for the streaming host."
  type        = string
  default     = "Standard_B2s"
}

variable "admin_username" {
  description = "Linux administrator username."
  type        = string
  default     = "azureuser"
}

variable "admin_source_cidr" {
  description = "CIDR allowed to connect to SSH, normally your public IP with /32."
  type        = string

  validation {
    condition     = can(cidrhost(var.admin_source_cidr, 0))
    error_message = "admin_source_cidr must be a valid CIDR, for example 203.0.113.10/32."
  }
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key installed on the VM."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "tags" {
  description = "Additional tags applied to Azure resources."
  type        = map(string)

  default = {
    project    = "rtsp-stream-console"
    managed_by = "terraform"
  }
}