package rest

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// NewDefaultClientConfig retrieves the client configuration from the process environment
// using the "k8s.io/client-go/tools/clientcmd" utilities
func NewDefaultClientConfig() (*rest.Config, error) {
	apiConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, fmt.Errorf("error building config: %v", err)
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error building config: %v", err)
	}

	mutateConfig(cfg)
	return cfg, nil
}

// BuildDefaultHostClient builds the default host client.
// It uses NewDefaultClientConfig for retrieving the client configuration.
func BuildDefaultHostClient() (client.Client, error) {
	cfg, err := NewDefaultClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error building config: %v", err)
	}

	return BuildClient(cfg)
}

func BuildClient(cfg *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	return client.New(cfg, client.Options{Scheme: scheme})
}

// BuildDefaultRESTMapper builds a RESTMapper from the default client configuration.
func BuildDefaultRESTMapper() (meta.RESTMapper, error) {
	cfg, err := NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	return BuildRESTMapper(cfg)
}

// BuildRESTMapper builds a RESTMapper from a given configuration.
func BuildRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	m, err := apiutil.NewDynamicRESTMapper(cfg, hc)
	if err != nil {
		return nil, err
	}
	return m, nil
}
