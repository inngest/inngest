output "server_id" {
  description = "ID of the created server"
  value       = hcloud_server.server.id
}

output "server_status" {
  description = "Status of the server"
  value       = hcloud_server.server.status
}

output "server_ipv4" {
  description = "IPv4 address of the server"
  value       = hcloud_primary_ip.ipv4.ip_address
}

output "server_ipv6" {
  description = "IPv6 address of the server"
  value       = hcloud_primary_ip.ipv6.ip_address
}

output "server_datacenter" {
  description = "Datacenter where the server is located"
  value       = hcloud_server.server.datacenter
}
