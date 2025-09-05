package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClient 创建 Kubernetes 客户端
func CreateClient() (kubernetes.Interface, error) {
	return CreateClientWithOptions("", "")
}

// CreateClientWithOptions 使用指定的 kubeconfig 和 context 创建 Kubernetes 客户端
func CreateClientWithOptions(kubeconfig, context string) (kubernetes.Interface, error) {
	// 首先尝试集群内配置
	config, err := rest.InClusterConfig()
	if err != nil {
		// 如果不在集群内，使用 kubeconfig 文件
		config, err = buildConfigFromFlagsWithOptions(kubeconfig, context)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %v", err)
		}
	}

	// 创建客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return clientset, nil
}

// buildConfigFromFlags 从命令行标志或默认位置构建配置
func buildConfigFromFlags() (*rest.Config, error) {
	return buildConfigFromFlagsWithOptions("", "")
}

// buildConfigFromFlagsWithOptions 使用指定的 kubeconfig 和 context 构建配置
func buildConfigFromFlagsWithOptions(kubeconfig, context string) (*rest.Config, error) {
	var loadingRules *clientcmd.ClientConfigLoadingRules

	if kubeconfig != "" {
		// 明确指定了 kubeconfig 文件路径
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: Using explicit kubeconfig path: %s\n", kubeconfig)
			fmt.Fprintf(os.Stderr, "Debug: Path length: %d characters\n", len(kubeconfig))
		}

		// 检查路径长度
		if len(kubeconfig) > 255 {
			return nil, fmt.Errorf("kubeconfig path too long (%d characters): %s", len(kubeconfig), kubeconfig)
		}

		// 检查文件是否存在
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig file not found at %s", kubeconfig)
		}

		loadingRules = &clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeconfig,
		}
	} else {
		// 使用默认的加载规则，这会自动处理：
		// 1. KUBECONFIG 环境变量（包括多个文件）
		// 2. 默认的 ~/.kube/config 文件
		// 3. 集群内配置
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: Using default kubeconfig loading rules\n")
		}
		loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()

		// 调试信息：显示将要加载的文件
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
				fmt.Fprintf(os.Stderr, "Debug: KUBECONFIG env var: %s\n", kubeconfigEnv)
				paths := filepath.SplitList(kubeconfigEnv)
				fmt.Fprintf(os.Stderr, "Debug: Will attempt to merge %d config files\n", len(paths))
				for i, path := range paths {
					fmt.Fprintf(os.Stderr, "Debug: Config file %d: %s\n", i+1, path)
				}
			} else {
				homeDir, _ := os.UserHomeDir()
				defaultPath := filepath.Join(homeDir, ".kube", "config")
				fmt.Fprintf(os.Stderr, "Debug: Will use default config: %s\n", defaultPath)
			}
		}
	}

	// 创建配置覆盖
	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: Using context: %s\n", context)
		}
	}

	// 创建客户端配置
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	// 构建配置
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
	}

	if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "Debug: Successfully created kubernetes config\n")
	}
	return config, nil
}

// getKubeconfigPath 获取 kubeconfig 文件路径
func getKubeconfigPath() string {
	// 首先检查 KUBECONFIG 环境变量
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: Found KUBECONFIG env var: %s\n", kubeconfigEnv)
		}

		// KUBECONFIG 可能包含多个用冒号分隔的路径
		// 在这种情况下，我们使用第一个存在的文件
		paths := filepath.SplitList(kubeconfigEnv)
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
					fmt.Fprintf(os.Stderr, "Debug: Using first valid path from KUBECONFIG: %s\n", path)
				}
				return path
			}
		}

		// 如果没有找到有效的文件，返回第一个路径（可能会导致错误，但这是预期的）
		if len(paths) > 0 {
			if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "Debug: No valid files found in KUBECONFIG, using first: %s\n", paths[0])
			}
			return paths[0]
		}
	}

	// 使用默认路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: Failed to get home directory: %v\n", err)
		}
		return ""
	}

	defaultPath := filepath.Join(homeDir, ".kube", "config")
	if os.Getenv("KUBECTL_NODE_MGR_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "Debug: Using default kubeconfig path: %s\n", defaultPath)
	}
	return defaultPath
}

// GetNamespace 获取当前命名空间
func GetNamespace() string {
	// 首先检查环境变量
	if ns := os.Getenv("KUBECTL_NAMESPACE"); ns != "" {
		return ns
	}

	// 尝试从 kubeconfig 获取当前上下文的命名空间
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath == "" {
		return "default"
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "default"
	}

	if config.CurrentContext == "" {
		return "default"
	}

	context, exists := config.Contexts[config.CurrentContext]
	if !exists || context.Namespace == "" {
		return "default"
	}

	return context.Namespace
}
