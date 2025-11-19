// Package kubernetes provides Kubernetes client implementations
package kubernetes

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// KubernetesClient provides Kubernetes API operations
type KubernetesClient interface {
	// CreateDeployment creates a Kubernetes deployment
	CreateDeployment(ctx context.Context, deployment *Deployment) error

	// GetDeployment gets a deployment by name
	GetDeployment(ctx context.Context, namespace string, name string) (*Deployment, error)

	// ListDeployments lists deployments in a namespace
	ListDeployments(ctx context.Context, namespace string) ([]*Deployment, error)

	// DeleteDeployment deletes a deployment
	DeleteDeployment(ctx context.Context, namespace string, name string) error

	// ListPods lists pods in a namespace
	ListPods(ctx context.Context, namespace string, labels map[string]string) ([]*Pod, error)

	// GetPodLogs gets logs from a pod
	GetPodLogs(ctx context.Context, namespace string, podName string, tailLines int) ([]string, error)

	// CreateService creates a Kubernetes service
	CreateService(ctx context.Context, service *Service) error

	// CreateConfigMap creates a Kubernetes ConfigMap
	CreateConfigMap(ctx context.Context, configMap *ConfigMap) error
}

// Deployment represents a Kubernetes deployment
type Deployment struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int
	Labels    map[string]string
	Env       map[string]string
	Ports     []int
}

// Pod represents a Kubernetes pod
type Pod struct {
	Name      string
	Namespace string
	Status    string
	Labels    map[string]string
	CreatedAt time.Time
}

// Service represents a Kubernetes service
type Service struct {
	Name      string
	Namespace string
	Type      string
	Ports     []ServicePort
	Selector  map[string]string
}

// ServicePort represents a service port
type ServicePort struct {
	Name       string
	Port       int
	TargetPort int
	Protocol   string
}

// ConfigMap represents a Kubernetes ConfigMap
type ConfigMap struct {
	Name      string
	Namespace string
	Data      map[string]string
}

// k8sClient implements KubernetesClient using client-go
type k8sClient struct {
	clientset  kubernetes.Interface
	timeout    time.Duration
}

// NewKubernetesClient creates a new Kubernetes client
func NewKubernetesClient(kubeconfig string, timeout time.Duration) (KubernetesClient, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig file
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
			}
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &k8sClient{
		clientset: clientset,
		timeout:   timeout,
	}, nil
}

// CreateDeployment creates a Kubernetes deployment
func (c *k8sClient) CreateDeployment(ctx context.Context, deployment *Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment cannot be nil")
	}
	if deployment.Name == "" {
		return fmt.Errorf("deployment name cannot be empty")
	}
	if deployment.Namespace == "" {
		deployment.Namespace = "default"
	}

	logger.Info("Creating Kubernetes deployment",
		zap.String("name", deployment.Name),
		zap.String("namespace", deployment.Namespace),
		zap.String("image", deployment.Image),
		zap.Int("replicas", deployment.Replicas),
	)

	// Build container ports
	containerPorts := []corev1.ContainerPort{}
	for _, port := range deployment.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: int32(port),
		})
	}

	// Build environment variables
	envVars := []corev1.EnvVar{}
	for key, value := range deployment.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Create deployment spec
	replicas := int32(deployment.Replicas)
	if replicas == 0 {
		replicas = 1
	}

	k8sDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels:    deployment.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: deployment.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deployment.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  deployment.Name,
							Image: deployment.Image,
							Ports: containerPorts,
							Env:   envVars,
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.AppsV1().Deployments(deployment.Namespace).Create(ctx, k8sDeployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	logger.Info("Kubernetes deployment created successfully",
		zap.String("name", deployment.Name),
		zap.String("namespace", deployment.Namespace),
	)

	return nil
}

// GetDeployment gets a deployment by name
func (c *k8sClient) GetDeployment(ctx context.Context, namespace string, name string) (*Deployment, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if namespace == "" {
		namespace = "default"
	}

	logger.Debug("Getting Kubernetes deployment",
		zap.String("name", name),
		zap.String("namespace", namespace),
	)

	k8sDeployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	replicas := 0
	if k8sDeployment.Spec.Replicas != nil {
		replicas = int(*k8sDeployment.Spec.Replicas)
	}

	ports := []int{}
	if len(k8sDeployment.Spec.Template.Spec.Containers) > 0 {
		for _, port := range k8sDeployment.Spec.Template.Spec.Containers[0].Ports {
			ports = append(ports, int(port.ContainerPort))
		}
	}

	env := make(map[string]string)
	if len(k8sDeployment.Spec.Template.Spec.Containers) > 0 {
		for _, envVar := range k8sDeployment.Spec.Template.Spec.Containers[0].Env {
			env[envVar.Name] = envVar.Value
		}
	}

	return &Deployment{
		Name:      k8sDeployment.Name,
		Namespace: k8sDeployment.Namespace,
		Image:     k8sDeployment.Spec.Template.Spec.Containers[0].Image,
		Replicas:  replicas,
		Labels:    k8sDeployment.Labels,
		Env:       env,
		Ports:     ports,
	}, nil
}

// ListDeployments lists deployments in a namespace
func (c *k8sClient) ListDeployments(ctx context.Context, namespace string) ([]*Deployment, error) {
	if namespace == "" {
		namespace = "default"
	}

	logger.Debug("Listing Kubernetes deployments",
		zap.String("namespace", namespace),
	)

	k8sDeployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]*Deployment, 0, len(k8sDeployments.Items))
	for _, k8sDeployment := range k8sDeployments.Items {
		replicas := 0
		if k8sDeployment.Spec.Replicas != nil {
			replicas = int(*k8sDeployment.Spec.Replicas)
		}

		ports := []int{}
		if len(k8sDeployment.Spec.Template.Spec.Containers) > 0 {
			for _, port := range k8sDeployment.Spec.Template.Spec.Containers[0].Ports {
				ports = append(ports, int(port.ContainerPort))
			}
		}

		env := make(map[string]string)
		if len(k8sDeployment.Spec.Template.Spec.Containers) > 0 {
			for _, envVar := range k8sDeployment.Spec.Template.Spec.Containers[0].Env {
				env[envVar.Name] = envVar.Value
			}
		}

		deployments = append(deployments, &Deployment{
			Name:      k8sDeployment.Name,
			Namespace: k8sDeployment.Namespace,
			Image:     k8sDeployment.Spec.Template.Spec.Containers[0].Image,
			Replicas:  replicas,
			Labels:    k8sDeployment.Labels,
			Env:       env,
			Ports:     ports,
		})
	}

	return deployments, nil
}

// DeleteDeployment deletes a deployment
func (c *k8sClient) DeleteDeployment(ctx context.Context, namespace string, name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if namespace == "" {
		namespace = "default"
	}

	logger.Info("Deleting Kubernetes deployment",
		zap.String("name", name),
		zap.String("namespace", namespace),
	)

	err := c.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	logger.Info("Kubernetes deployment deleted successfully",
		zap.String("name", name),
		zap.String("namespace", namespace),
	)

	return nil
}

// ListPods lists pods in a namespace
func (c *k8sClient) ListPods(ctx context.Context, namespace string, labelMap map[string]string) ([]*Pod, error) {
	if namespace == "" {
		namespace = "default"
	}

	logger.Debug("Listing Kubernetes pods",
		zap.String("namespace", namespace),
	)

	opts := metav1.ListOptions{}
	if len(labelMap) > 0 {
		selector := labels.SelectorFromSet(labelMap)
		opts.LabelSelector = selector.String()
	}

	k8sPods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	pods := make([]*Pod, 0, len(k8sPods.Items))
	for _, k8sPod := range k8sPods.Items {
		pods = append(pods, &Pod{
			Name:      k8sPod.Name,
			Namespace: k8sPod.Namespace,
			Status:    string(k8sPod.Status.Phase),
			Labels:    k8sPod.Labels,
			CreatedAt: k8sPod.CreationTimestamp.Time,
		})
	}

	return pods, nil
}

// GetPodLogs gets logs from a pod
func (c *k8sClient) GetPodLogs(ctx context.Context, namespace string, podName string, tailLines int) ([]string, error) {
	if podName == "" {
		return nil, fmt.Errorf("podName cannot be empty")
	}
	if namespace == "" {
		namespace = "default"
	}
	if tailLines <= 0 {
		tailLines = 100
	}

	logger.Debug("Getting Kubernetes pod logs",
		zap.String("pod", podName),
		zap.String("namespace", namespace),
		zap.Int("tail_lines", tailLines),
	)

	opts := &corev1.PodLogOptions{
		TailLines: func() *int64 { lines := int64(tailLines); return &lines }(),
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer stream.Close()

	logBytes, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read pod logs: %w", err)
	}

	// Split logs by newline
	logLines := []string{}
	currentLine := ""
	for _, b := range logBytes {
		if b == '\n' {
			if currentLine != "" {
				logLines = append(logLines, currentLine)
				currentLine = ""
			}
		} else {
			currentLine += string(b)
		}
	}
	if currentLine != "" {
		logLines = append(logLines, currentLine)
	}

	return logLines, nil
}

// CreateService creates a Kubernetes service
func (c *k8sClient) CreateService(ctx context.Context, service *Service) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}
	if service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if service.Namespace == "" {
		service.Namespace = "default"
	}

	logger.Info("Creating Kubernetes service",
		zap.String("name", service.Name),
		zap.String("namespace", service.Namespace),
		zap.String("type", service.Type),
	)

	// Build service ports
	servicePorts := []corev1.ServicePort{}
	for _, port := range service.Ports {
		targetPort := int32(port.TargetPort)
		if targetPort == 0 {
			targetPort = int32(port.Port)
		}
		protocol := corev1.ProtocolTCP
		if port.Protocol != "" {
			protocol = corev1.Protocol(port.Protocol)
		}
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Port:       int32(port.Port),
			TargetPort: intstr.FromInt32(targetPort),
			Protocol:   protocol,
		})
	}

	serviceType := corev1.ServiceTypeClusterIP
	if service.Type != "" {
		serviceType = corev1.ServiceType(service.Type)
	}

	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     serviceType,
			Selector: service.Selector,
			Ports:    servicePorts,
		},
	}

	_, err := c.clientset.CoreV1().Services(service.Namespace).Create(ctx, k8sService, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	logger.Info("Kubernetes service created successfully",
		zap.String("name", service.Name),
		zap.String("namespace", service.Namespace),
	)

	return nil
}

// CreateConfigMap creates a Kubernetes ConfigMap
func (c *k8sClient) CreateConfigMap(ctx context.Context, configMap *ConfigMap) error {
	if configMap == nil {
		return fmt.Errorf("configMap cannot be nil")
	}
	if configMap.Name == "" {
		return fmt.Errorf("configMap name cannot be empty")
	}
	if configMap.Namespace == "" {
		configMap.Namespace = "default"
	}

	logger.Info("Creating Kubernetes ConfigMap",
		zap.String("name", configMap.Name),
		zap.String("namespace", configMap.Namespace),
	)

	k8sConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMap.Name,
			Namespace: configMap.Namespace,
		},
		Data: configMap.Data,
	}

	_, err := c.clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(ctx, k8sConfigMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	logger.Info("Kubernetes ConfigMap created successfully",
		zap.String("name", configMap.Name),
		zap.String("namespace", configMap.Namespace),
	)

	return nil
}
