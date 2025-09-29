variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

variable "server_name" {
  description = "Name of the server"
  type        = string
}

variable "server_location" {
  description = "Location of the server (default: eu-central)"
  type        = string
  default     = "eu-central"
}

variable "server_image" {
  description = "Image to use for the server"
  type        = string
  default     = "ubuntu-24.04"
}

variable "ssh_key_path" {
  description = "Path to the SSH public key file"
  type        = string
  default     = "~/.ssh/id_rsa_ansible.pub"
}
