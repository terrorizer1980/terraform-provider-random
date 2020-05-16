module github.com/terraform-providers/terraform-provider-random

replace (
	github.com/hashicorp/go-plugin v1.2.2 => ../go-plugin
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.0.0-20200501181702-56d861ef9928 => ../terraform-plugin-sdk
	github.com/hashicorp/terraform-plugin-test v1.3.0 => ../terraform-plugin-test
)

go 1.14

require (
	github.com/dustinkirkland/golang-petname v0.0.0-20170105215008-242afa0b4f8a
	github.com/hashicorp/errwrap v1.0.0
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.0.0-20200501181702-56d861ef9928
)
