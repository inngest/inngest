terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.44.1"
    }
  }
  required_version = ">= 1.0.0"
}

provider "hcloud" {
  token = var.hcloud_token
}
