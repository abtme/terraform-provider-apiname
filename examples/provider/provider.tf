terraform {
  required_providers {
    apiname = {
      source = "abtme/apiname"
    }
  }
}

# reseller_id/api_key can also be set via the APINAME_RESELLER_ID/APINAME_API_KEY
# environment variables instead of hardcoding them here.
provider "apiname" {
  reseller_id = "your-reseller-id"
  api_key     = "your-api-key"
}
