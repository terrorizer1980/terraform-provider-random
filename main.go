package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/terraform-providers/terraform-provider-random/random"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debuggable", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if debugMode {
		ctx, cancel := context.WithCancel(context.Background())
		config, closeCh, err := plugin.DebugServe(ctx, &plugin.ServeOpts{
			ProviderFunc: random.Provider,
		})
		if err != nil {
			fmt.Printf("Error launching debug server: %s\n", err.Error())
		}
		// Ctrl-C will stop the server
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		defer func() {
			signal.Stop(sigCh)
			cancel()
		}()
		go func() {
			select {
			case <-sigCh:
				cancel()
			case <-ctx.Done():
			}
		}()
		reattachStr, err := json.Marshal(map[string]plugin.ReattachConfig{
			"hashicorp/random": config,
		})
		if err != nil {
			fmt.Printf("Error building reattach string: %s\n", err.Error())
			cancel()
			<-closeCh
			os.Exit(1)
		}

		fmt.Printf("Provider server started; to attach Terraform, set TF_REATTACH_PROVIDERS to the following:\n%s\n", string(reattachStr))

		// wait for the server to be done
		<-closeCh
		return
	}
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: random.Provider})
}
