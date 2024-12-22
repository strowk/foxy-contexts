package main

import "k8s.io/client-go/tools/clientcmd"

// --8<-- [start:dependency_create]

// NewK8sClientConfig creates a new k8s client config
// , which we define separately in order to give it to
// fx to later inject it into potentially several tools as a dependency
func NewK8sClientConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
}

// --8<-- [end:dependency_create]
