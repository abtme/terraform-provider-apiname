resource "apiname_domain_nameservers" "example" {
  domain_name = "example.com"

  nameservers = [
    "ns1.yourdns.com",
    "ns2.yourdns.com",
  ]
}
