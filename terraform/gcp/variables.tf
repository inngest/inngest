variable "project_id" {
  description = "ID do projeto no GCP"
  type        = string
}

variable "region" {
  description = "Região no GCP"
  type        = string
}

variable "zone" {
  description = "Zona no GCP"
  type        = string
}

variable "service_account" {
  description = "Conta de serviço a ser utilizada para a VM"
  type        = string
}

variable "ssh_user" {
  description = "Usuário SSH para acesso à VM"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Caminho para a chave privada SSH (ex: ~/.ssh/id_rsa_ansible)"
  type        = string
}

variable "ssh_public_key_path" {
  description = "Caminho para a chave pública SSH (ex: ~/.ssh/id_rsa_ansible.pub)"
  type        = string
}
