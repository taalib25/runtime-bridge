terraform {
  backend "s3" {
    bucket = "hermes-tofu-state"
    key    = "clusters/kh-test/tofu.tfstate"
    region = "auto"

    # endpoint injected at tofu init via -backend-config="endpoint=..."
    # credentials injected via AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars

    use_lockfile    = true
    use_path_style  = true  # required for Cloudflare R2 (force_path_style deprecated in 1.10)

    skip_credentials_validation = true
    skip_region_validation      = true
    skip_metadata_api_check     = true
  }
}
