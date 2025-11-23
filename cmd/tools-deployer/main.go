package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/tools/deployers"
)

func main() {
	var deployType, projectName, projectPath, namespace, image, manifestsPath, provider string
	var replicas int
	var labels, env string
	var ports string
	var kubeconfig string
	var help bool

	flag.StringVar(&deployType, "type", "", "Deployment type: kubernetes, docker, serverless, hybrid")
	flag.StringVar(&projectName, "name", "", "Project name (required)")
	flag.StringVar(&projectPath, "path", "", "Project path (required)")
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	flag.StringVar(&image, "image", "", "Container image")
	flag.StringVar(&manifestsPath, "manifests", "", "Path to Kubernetes manifests")
	flag.IntVar(&replicas, "replicas", 1, "Number of replicas")
	flag.StringVar(&labels, "labels", "", "Comma-separated key=value labels")
	flag.StringVar(&env, "env", "", "Comma-separated key=value environment variables")
	flag.StringVar(&ports, "ports", "", "Comma-separated list of ports")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	flag.StringVar(&provider, "provider", "aws", "Serverless provider (aws/azure/gcp)")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()

	if help || deployType == "" || projectName == "" || projectPath == "" {
		usage()
		if deployType == "" || projectName == "" || projectPath == "" {
			os.Exit(1)
		}
	}

	ctx := context.Background()

	var result interface{}
	var err error

	switch deployType {
	case "kubernetes":
		if image == "" && manifestsPath == "" {
			fmt.Fprintf(os.Stderr, "Error: image or manifests path is required\n")
			os.Exit(1)
		}
		req := deployers.KubernetesDeployRequest{
			ProjectName:   projectName,
			ProjectPath:   projectPath,
			Namespace:     namespace,
			Image:         image,
			Replicas:      replicas,
			ManifestsPath: manifestsPath,
		}
		req.Labels = parseKeyValue(labels)
		req.Env = parseKeyValue(env)
		req.Ports = parsePorts(ports)

		deployer, err := deployers.NewKubernetesDeployer(kubeconfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes deployer: %v\n", err)
			os.Exit(1)
		}
		result, err = deployer.Deploy(ctx, req)

	case "docker":
		req := deployers.DockerDeployRequest{
			ProjectName: projectName,
			ProjectPath: projectPath,
			ImageName:   image,
		}
		req.Env = parseKeyValue(env)
		// Convert ports to map[string]string format expected by DockerDeployRequest
		portsMap := make(map[string]string)
		for _, p := range parsePorts(ports) {
			portsMap[fmt.Sprintf("%d", p)] = fmt.Sprintf("%d", p)
		}
		req.Ports = portsMap

		deployer, err := deployers.NewDockerDeployer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Docker deployer: %v\n", err)
			os.Exit(1)
		}
		result, err = deployer.Deploy(ctx, req)

	case "serverless":
		req := deployers.ServerlessDeployRequest{
			FunctionName: projectName,
			FunctionPath: projectPath,
			Provider:     provider,
			Runtime:      "go",
			Handler:      "main",
		}
		req.Environment = parseKeyValue(env)

		deployer, err := deployers.NewServerlessDeployer(provider)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Serverless deployer: %v\n", err)
			os.Exit(1)
		}
		result, err = deployer.Deploy(ctx, req)

	case "hybrid":
		// Hybrid deployer doesn't exist yet, use Kubernetes as fallback
		fmt.Fprintf(os.Stderr, "Error: Hybrid deployment not yet implemented, using Kubernetes\n")
		req := deployers.KubernetesDeployRequest{
			ProjectName: projectName,
			ProjectPath: projectPath,
			Namespace:   "default",
			Image:       image,
			Replicas:    replicas,
		}
		req.Labels = parseKeyValue(labels)
		req.Env = parseKeyValue(env)
		req.Ports = parsePorts(ports)

		deployer, err := deployers.NewKubernetesDeployer(kubeconfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes deployer: %v\n", err)
			os.Exit(1)
		}
		result, err = deployer.Deploy(ctx, req)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown deployment type: %s\n", deployType)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output result as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s -type TYPE -name NAME -path PATH [OPTIONS]

Deployment types:
  kubernetes - Deploy to Kubernetes
  docker     - Deploy using Docker
  serverless - Deploy to serverless platform
  hybrid     - Deploy using hybrid approach

Options:
  -type TYPE        Deployment type (required)
  -name NAME        Project name (required)
  -path PATH        Project path (required)
  -namespace NS     Kubernetes namespace (default: default)
  -image IMAGE      Container image
  -manifests PATH   Path to Kubernetes manifests
  -replicas N       Number of replicas (default: 1)
  -labels LIST      Comma-separated key=value labels
  -env LIST         Comma-separated key=value environment variables
  -ports LIST       Comma-separated list of ports
  -kubeconfig PATH  Path to kubeconfig file
  -help             Show this help

Examples:
  %s -type kubernetes -name my-app -path ./my-app -image my-app:latest -namespace prod
  %s -type docker -name my-app -path ./my-app -image my-app:latest
`, os.Args[0], os.Args[0], os.Args[0])
}

func parseKeyValue(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}
	for _, pair := range strings.Split(s, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

func parsePorts(s string) []int {
	var result []int
	if s == "" {
		return result
	}
	for _, portStr := range strings.Split(s, ",") {
		var port int
		if _, err := fmt.Sscanf(strings.TrimSpace(portStr), "%d", &port); err == nil {
			result = append(result, port)
		}
	}
	return result
}
