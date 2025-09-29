resource "hcloud_ssh_key" "default" {
  name       = "${var.server_name}-ansible-key"
  public_key = file(var.ssh_key_path)
}

resource "hcloud_primary_ip" "ipv4" {
  name          = "${var.server_name}-ipv4"
  type          = "ipv4"
  assignee_type = "server"
  auto_delete   = true
  datacenter    = "fsn1-dc14"
}

resource "hcloud_primary_ip" "ipv6" {
  name          = "${var.server_name}-ipv6"
  type          = "ipv6"
  assignee_type = "server"
  auto_delete   = true
  datacenter    = "fsn1-dc14"
}

resource "hcloud_server" "server" {
  name        = var.server_name
  server_type = "cx32"
  image       = var.server_image
  location    = var.server_location
  ssh_keys    = [hcloud_ssh_key.default.id]

  public_net {
    ipv4_enabled = true
    ipv4 = hcloud_primary_ip.ipv4.id
    ipv6_enabled = true
    ipv6 = hcloud_primary_ip.ipv6.id
  }

  lifecycle {
    ignore_changes = [
      ssh_keys,
    ]
  }
}

resource "local_file" "ansible_inventory" {
  filename = "${path.module}/../../ansible/inventory"
  content  = <<-EOT
[servers]
${hcloud_primary_ip.ipv4.ip_address} ansible_user=root ansible_ssh_private_key_file=${replace(var.ssh_key_path, ".pub", "")}
EOT
}
