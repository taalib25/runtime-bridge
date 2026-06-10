terraform {
  backend "s3" {
    bucket = "hermes-tofu-state"
    key    = "clusters/kh-test/tofu.tfstate"
    region = "auto"

    # endpoint is injected at tofu init time via -backend-config=endpoint=...
    # or TF_BACKEND_CONFIG_endpoint env var in CI.

    use_lockfile = true

    skip_credentials_validation = true
    skip_region_validation      = true
    skip_metadata_api_check     = true
  }
}
