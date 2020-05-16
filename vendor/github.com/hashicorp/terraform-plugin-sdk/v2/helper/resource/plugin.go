package resource

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	grpcplugin "github.com/hashicorp/terraform-plugin-sdk/v2/internal/helper/plugin"
	proto "github.com/hashicorp/terraform-plugin-sdk/v2/internal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	tftest "github.com/hashicorp/terraform-plugin-test"
)

func runProviderCommand(f func() error, wd *tftest.WorkingDir, opts *plugin.ServeOpts) error {
	// offer an opt-out that runs tests in separate provider processes
	// this will behave just like prod
	if os.Getenv("TF_TEST_PROVIDERS_OOP") != "" {
		return f()
	}

	// the provider name is technically supposed to be specified
	// in the format returned by addrs.Provider.GetDisplay(), but
	// 1. I'm not importing the entire addrs package for this and
	// 2. we only get the provider name here. Fortunately, when
	// only a provider name is specified in a provider block--which
	// is how the config file we generate does things--Terraform
	// just automatically assumes it's in the hashicorp namespace
	// and the default registry.terraform.io host, so we can just
	// construct the output of GetDisplay() ourselves, based on the
	// provider name. GetDisplay() omits the default host, so for
	// our purposes this will always be hashicorp/PROVIDER_NAME.
	providerName := wd.GetHelper().GetPluginName()

	// providerName gets returned as terraform-provider-foo, and we
	// need just foo. So let's fix that.
	providerName = strings.TrimPrefix(providerName, "terraform-provider-")

	// We need to tell the provider which version of the Terraform
	// protocol to serve. Usually this is negotiated with Terraform
	// during the handshake that sets the server up, but because
	// we're manually setting the server up, it's on us to do.
	// Because the SDK only supports 0.12+ of Terraform at the
	// moment, we can just set this to 5 (the latest version of the
	// protocol) and call it a day. But if and when we get a version
	// 6, we're going to have to figure something out.
	protoVersion := 5 // TODO: make this configurable?

	// by default, run tests in the same process as the test runner
	// using the reattach behavior in Terraform. This ensures we get
	// test coverage and enables the use of delve as a debugger.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reattachCh := make(chan *goplugin.ReattachConfig)

	closeCh := make(chan struct{})
	opts.TestConfig = &goplugin.ServeTestConfig{
		Context:          ctx,
		ReattachConfigCh: reattachCh,
		CloseCh:          closeCh,
	}
	if opts.Logger == nil {
		opts.Logger = hclog.New(&hclog.LoggerOptions{
			Name:   "plugintest",
			Level:  hclog.Trace,
			Output: ioutil.Discard,
		})
	}

	os.Setenv("PLUGIN_PROTOCOL_VERSIONS", "5")
	go plugin.Serve(opts)
	var config *goplugin.ReattachConfig
	select {
	case config = <-reattachCh:
	case <-time.After(2 * time.Second):
		return errors.New("timeout waiting on reattach config")
	}

	if config == nil {
		return errors.New("nil reattach config received")
	}
	reattachStr := fmt.Sprintf("hashicorp/%s=%d|%s|%s|%s|%d|test",
		providerName,
		protoVersion,
		config.Addr.Network(),
		config.Addr.String(),
		config.Protocol,
		config.Pid,
	)
	wd.Setenv("TF_PROVIDER_REATTACH", reattachStr)
	err := f()
	if err != nil {
		log.Printf("[WARN] Got error running Terraform: %s", err)
	}
	cancel()
	<-closeCh

	// once we've run the Terraform command, let's remove the
	// reattach information from the WorkingDir's environment. The
	// WorkingDir will persist until the next call, but the server
	// in the reattach info doesn't exist anymore at this point, so
	// the reattach info is no longer valid. In theory it should be
	// overwritten in the next call, but just to avoid any
	// confusing bug reports, let's just unset the environment
	// variable altogether.
	wd.Unsetenv("TF_PROVIDER_REATTACH")

	// return any error returned from the orchestration code running
	// Terraform commands
	return err
}

// defaultPluginServeOpts builds ths *plugin.ServeOpts that you usually want to
// use when running runProviderCommand. It just sets the ProviderFunc to return
// the provider under test.
func defaultPluginServeOpts(wd *tftest.WorkingDir, providers map[string]*schema.Provider) *plugin.ServeOpts {
	return &plugin.ServeOpts{
		ProviderFunc: acctest.TestProviderFunc,
		GRPCProviderFunc: func() proto.ProviderServer {
			return grpcplugin.NewGRPCProviderServer(acctest.TestProviderFunc())
		},
	}
}
