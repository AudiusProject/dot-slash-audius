package orchestration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/AudiusProject/audius-d/conf"
	"github.com/joho/godotenv"
)

type OverrideEnv = map[string]string

// deploys a server node regardless of type
func RunNode(config conf.ContextConfig, serverConfig conf.BaseServerConfig, override OverrideEnv, containerName string, nodeType string, internalVolumes []string) error {
	imageTag := fmt.Sprintf("audius/audius-docker-compose:%s", config.Network.Tag)
	externalVolume := fmt.Sprintf("audius-d-%s", containerName)
	httpPorts := PortMapping(serverConfig.ExternalHttpPort, serverConfig.InternalHttpPort)
	httpsPorts := PortMapping(serverConfig.ExternalHttpsPort, serverConfig.InternalHttpsPort)
	formattedInternalVolumes := " -v " + strings.Join(internalVolumes, " -v ")

	// assemble wrapper command and run
	// todo: handle https port
	upCmd := fmt.Sprintf("docker run --privileged -d -v %s:/var/lib/docker %s %s --name %s %s %s", externalVolume, httpPorts, httpsPorts, containerName, formattedInternalVolumes, imageTag)
	if err := Sh(upCmd); err != nil {
		return err
	}

	// initialize override.env file
	localOverridePath := fmt.Sprintf("./%s-override.env", containerName)
	if err := godotenv.Write(override, localOverridePath); err != nil {
		return err
	}

	envCmd := fmt.Sprintf("docker cp %s %s:/root/audius-docker-compose/%s/override.env", localOverridePath, containerName, nodeType)
	if err := Sh(envCmd); err != nil {
		return err
	}

	cmd := fmt.Sprintf(`docker exec %s sh -c "while ! docker ps &> /dev/null; do echo 'starting up' && sleep 1; done"`, containerName)
	if err := runCommand("/bin/sh", "-c", cmd); err != nil {
		return err
	}

	if err := os.Remove(localOverridePath); err != nil {
		return err
	}

	// assemble inner command and run
	startCmd := fmt.Sprintf(`docker exec %s sh -c "cd %s && docker compose up -d"`, containerName, nodeType)
	if err := Sh(startCmd); err != nil {
		return err
	}

	return nil
}

func PortMapping(external, internal uint) string {
	return fmt.Sprintf("-p %d:%d", external, internal)
}

func Sh(cmd string) error {
	fmt.Println(cmd)
	return runCommand("/bin/sh", "-c", cmd)
}

func startDevnetDocker() {
	fmt.Println("Starting local eth, sol, and acdc chains")
	runCommand("docker", "compose", "-f", "./devnet/docker-compose.yml", "up", "-d")
}

func downDevnetDocker() {
	runCommand("docker", "compose", "-f", "./devnet/docker-compose.yml", "down")
}

func audiusCli(args ...string) {
	nodeType := ""
	audCli := []string{"exec", nodeType, "audius-cli"}
	cmds := append(audCli, args...)
	err := runCommand("docker", cmds...)
	if err != nil {
		log.Fatal(err)
	}
}

func dockerExec(arg ...string) string {
	nodeType := ""
	baseCmd := []string{"exec", nodeType}
	cmds := append(baseCmd, arg...)
	out, err := exec.Command("docker", cmds...).Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}

// generates relevant nethermind chain configuration files
// logic ported over from audius-docker-compose https://github.com/AudiusProject/audius-docker-compose/blob/stage/audius-cli#L848
func configureChainSpec(nodeType string, network string) {
	extraVanity := "0x22466c6578692069732061207468696e6722202d204166726900000000000000"
	extraSeal := "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	// gather env config
	// discovery-provider/stage.env for example
	networkEnvPath := fmt.Sprintf("discovery-provider/%s.env", network)
	networkEnv := dockerExec("cat", networkEnvPath)
	networkEnvMap, err := godotenv.Unmarshal(networkEnv)
	if err != nil {
		log.Fatal(err)
	}

	signers := networkEnvMap["audius_genesis_signers"]
	extraData := fmt.Sprintf("%s%s%s", extraVanity, signers, extraSeal)

	specTemplatePath := fmt.Sprintf("discovery-provider/chain/%s_spec_template.json", network)
	specInput := dockerExec("cat", specTemplatePath)
	var specData map[string]interface{}
	err = json.Unmarshal([]byte(specInput), &specData)
	if err != nil {
		log.Fatal(err)
	}

	networkId := specData["params"].(map[string]interface{})["networkID"].(string)
	fmt.Printf("Network id: %s\n", networkId)

	specData["genesis"].(map[string]interface{})["extraData"] = extraData

	specOutput, err := json.MarshalIndent(specData, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	peersStr := networkEnvMap["audius_static_nodes"]
	peers := strings.Split(peersStr, ",")
	peersOutput, err := json.MarshalIndent(peers, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("spec.json", specOutput, 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("static-nodes.json", peersOutput, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// docker cp ./spec.json creator-node:/root/audius-docker-compose/discovery-provider/chain
	err = exec.Command("docker", "cp", "./spec.json", fmt.Sprintf("%s:/root/audius-docker-compose/discovery-provider/chain", nodeType)).Run()
	if err != nil {
		log.Fatal(err)
	}

	err = exec.Command("docker", "cp", "./static-nodes.json", fmt.Sprintf("%s:/root/audius-docker-compose/discovery-provider/chain", nodeType)).Run()
	if err != nil {
		log.Fatal(err)
	}

	// cleanup, remove temp files from filesystem
	err = os.Remove("spec.json")
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove("static-nodes.json")
	if err != nil {
		log.Fatal(err)
	}
}
