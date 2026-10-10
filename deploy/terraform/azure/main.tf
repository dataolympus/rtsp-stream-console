locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    var.tags,
    {
      environment = var.environment
    }
  )
}

resource "azurerm_resource_group" "main" {
  name     = "rg-${local.name_prefix}"
  location = var.location

  tags = local.common_tags
}

resource "azurerm_virtual_network" "main" {
  name                = "vnet-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  address_space = [
    "10.20.0.0/16",
  ]

  tags = local.common_tags
}

resource "azurerm_subnet" "main" {
  name                 = "snet-app"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name

  address_prefixes = [
    "10.20.1.0/24",
  ]
}

resource "azurerm_network_security_group" "main" {
  name                = "nsg-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  tags = local.common_tags
}

resource "azurerm_network_security_rule" "ssh" {
  name                       = "allow-ssh-admin"
  priority                   = 100
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Tcp"
  source_port_range          = "*"
  destination_port_range     = "22"
  source_address_prefix      = var.admin_source_cidr
  destination_address_prefix = "*"

  resource_group_name         = azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.main.name
}

resource "azurerm_network_security_rule" "http" {
  name                       = "allow-http"
  priority                   = 110
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Tcp"
  source_port_range          = "*"
  destination_port_range     = "80"
  source_address_prefix      = "Internet"
  destination_address_prefix = "*"

  resource_group_name         = azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.main.name
}

resource "azurerm_network_security_rule" "https" {
  name                       = "allow-https"
  priority                   = 120
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Tcp"
  source_port_range          = "*"
  destination_port_range     = "443"
  source_address_prefix      = "Internet"
  destination_address_prefix = "*"

  resource_group_name         = azurerm_resource_group.main.name
  network_security_group_name = azurerm_network_security_group.main.name
}

resource "azurerm_subnet_network_security_group_association" "main" {
  subnet_id                 = azurerm_subnet.main.id
  network_security_group_id = azurerm_network_security_group.main.id
}

resource "azurerm_public_ip" "main" {
  name                = "pip-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  allocation_method       = "Static"
  sku                     = "Standard"
  idle_timeout_in_minutes = 15

  tags = local.common_tags
}

resource "azurerm_network_interface" "main" {
  name                = "nic-${local.name_prefix}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  ip_configuration {
    name = "primary"

    subnet_id = azurerm_subnet.main.id

    private_ip_address_allocation = "Dynamic"

    public_ip_address_id = azurerm_public_ip.main.id
  }

  tags = local.common_tags
}

resource "azurerm_linux_virtual_machine" "main" {
  name                = "vm-${local.name_prefix}"
  computer_name       = "rtsp-stream"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  size                = var.vm_size

  admin_username                  = var.admin_username
  disable_password_authentication = true

  network_interface_ids = [
    azurerm_network_interface.main.id,
  ]

  admin_ssh_key {
    username = var.admin_username

    public_key = file(
      pathexpand(var.ssh_public_key_path)
    )
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "StandardSSD_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  custom_data = base64encode(
    templatefile(
      "${path.module}/cloud-init.yaml.tftpl",
      {
        admin_username = var.admin_username
      }
    )
  )

  tags = local.common_tags
}