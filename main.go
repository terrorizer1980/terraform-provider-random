package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/terraform-providers/terraform-provider-random/random"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debuggable", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/hashicorp/random", &plugin.ServeOpts{
			ProviderFunc: random.Provider,
		})
		if err != nil {
			log.Println("Error starting server:", err)
			os.Exit(1)
		}
	} else {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: random.Provider})
	}
}
