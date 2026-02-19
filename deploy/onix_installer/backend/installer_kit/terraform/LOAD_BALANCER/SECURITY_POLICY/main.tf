resource "google_compute_security_policy" "onix_security_policy" {
  name        = "onix-security-policy-${var.app_name}"
  description = "Cloud Armor policy for ONDC Seller Dev"

  # Rule 1: Geo-Location (Priority 1000)
  rule {
    action   = "deny(403)"
    priority = "1000"
    match {
      expr {
        expression = "!(${join(" || ", [for r in var.allowed_regions : "origin.region_code == '${r}'"])})"
      }
    }
    description = "Block unauthorized regions"
  }

  # Rule 2: IP Rate Limiting (Priority 2000)
  rule {
    action   = "throttle"
    priority = "2000"
    match {
      versioned_expr = "SRC_IPS_V1"
      config { src_ip_ranges = ["*"] }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      rate_limit_threshold {
        count        = var.rate_limit_count
        interval_sec = 60
      }
      enforce_on_key = "IP"
    }
  }

  # Rule 3: OWASP Top 10 (Priority 3000)
  rule {
    action   = "deny(403)"
    priority = "3000"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-stable') || evaluatePreconfiguredExpr('xss-stable')"
      }
    }
  }

  # Rule 4: Default Allow
  rule {
    action   = "allow"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config { src_ip_ranges = ["*"] }
    }
  }
}

