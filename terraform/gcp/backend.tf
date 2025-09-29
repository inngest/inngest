terraform {
  backend "gcs" {
    bucket = "terraform-states-83193-tfstate"
    prefix = "shared-services-evolution-api/state"
  }
}
