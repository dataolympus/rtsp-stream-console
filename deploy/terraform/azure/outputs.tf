output "resource_group_name" {
  description = "Azure resource group containing the deployment."
  value       = azurerm_resource_group.main.name
}

output "public_ip_address" {
  description = "Static public IPv4 address assigned to the application host."
  value       = azurerm_public_ip.main.ip_address
}

output "network_interface_id" {
  description = "Network interface that will be attached to the application VM."
  value       = azurerm_network_interface.main.id
}