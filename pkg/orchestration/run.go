package orchestration

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/AudiusProject/audius-d/pkg/conf"
	"github.com/AudiusProject/audius-d/pkg/register"
)

func StartDevnet(_ *conf.ContextConfig) {
	startDevnetDocker()
}

func DownDevnet(_ *conf.ContextConfig) {
	downDevnetDocker()
}

func RunAudiusWithConfig(config *conf.ContextConfig) {
	if config.Network.Devnet {
		startDevnetDocker()
		// gated on devnet for safety right now
		registerDevnetNodes(config)
	}

	dashboardVolume := "/dashboard-dist:/dashboard-dist"
	esDataVolume := "/esdata:/esdata"

	// mac local volumes need some extra stuff
	// stick into /var/k8s as if these existed then
	if runtime.GOOS == "darwin" {
		esDataVolume = "/var/k8s/esdata:/esdata"
		dashboardVolume = "/var/k8s/dashboard-dist:/dashboard-dist"
	}

	for cname, cc := range config.CreatorNodes {
		creatorVolumes := []string{"/var/k8s/mediorum:/var/k8s/mediorum", "/var/k8s/creator-node-backend:/var/k8s/creator-node-backend", "/var/k8s/creator-node-db:/var/k8s/creator-node-db", "/var/k8s/bolt:/var/k8s/bolt", dashboardVolume}
		override := cc.ToOverrideEnv(config.Network)
		RunNode(config.Network, cc.BaseServerConfig, override, cname, "creator-node", creatorVolumes)
		if cc.AwaitHealthy {
			awaitHealthy(cname, cc.Host, cc.ExternalHttpPort)
		}
	}
	for cname, dc := range config.DiscoveryNodes {
		discoveryVolumes := []string{"/var/k8s/discovery-provider-db:/var/k8s/discovery-provider-db", "/var/k8s/discovery-provider-chain:/var/k8s/discovery-provider-chain", "/var/k8s/bolt:/var/k8s/bolt", esDataVolume, dashboardVolume}
		override := dc.ToOverrideEnv(config.Network)
		RunNode(config.Network, dc.BaseServerConfig, override, cname, "discovery-provider", discoveryVolumes)
		// discovery requires a few extra things
		if !config.Network.Devnet {
			audiusCli(cname, "launch-chain")
		}
		if dc.AwaitHealthy {
			awaitHealthy(cname, dc.Host, dc.ExternalHttpPort)
		}
	}
	for cname, id := range config.IdentityService {
		identityVolumes := []string{"/var/k8s/identity-service-db:/var/lib/postgresql/data"}
		override := id.ToOverrideEnv(config.Network)
		RunNode(config.Network, id.BaseServerConfig, override, cname, "identity-service", identityVolumes)
		if id.AwaitHealthy {
			awaitHealthy(cname, id.Host, id.ExternalHttpPort)
		}
	}
}

func RunDown(config *conf.ContextConfig) {
	// easiest way
	cnames := []string{"rm", "-f"}

	for cname := range config.CreatorNodes {
		cnames = append(cnames, cname)
	}
	for cname := range config.DiscoveryNodes {
		cnames = append(cnames, cname)
	}
	for cname := range config.IdentityService {
		cnames = append(cnames, cname)
	}
	runCommand("docker", cnames...)
	if config.Network.Devnet {
		downDevnetDocker()
	}
}

func registerDevnetNodes(config *conf.ContextConfig) {
	for _, cc := range config.CreatorNodes {
		if cc.Register {
			register.RegisterNode(
				"content-node",
				cc.Host,
				"http://localhost:8546",
				config.Network.EthTokenAddress,
				config.Network.EthContractsRegistryAddress,
				cc.OperatorWallet,
				cc.OperatorPrivateKey,
			)
		}
	}
	fmt.Println("content nodes registered")
	for _, dc := range config.DiscoveryNodes {
		if dc.Register {
			register.RegisterNode(
				"discovery-provider",
				dc.Host,
				"http://localhost:8546",
				config.Network.EthTokenAddress,
				config.Network.EthContractsRegistryAddress,
				dc.OperatorWallet,
				dc.OperatorPrivateKey,
			)
		}
	}
	fmt.Println("discovery providers registered")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
