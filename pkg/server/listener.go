package server

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type WorkloadClusterListener struct {
	host string
	port int

	scheme *runtime.Scheme

	apiServers                  sets.Set[string]
	apiServerCaCertificate      *x509.Certificate
	apiServerCaKey              *rsa.PrivateKey
	apiServerServingCertificate *tls.Certificate

	adminCertificate *x509.Certificate
	adminKey         *rsa.PrivateKey

	etcdMembers             sets.Set[string]
	etcdServingCertificates map[string]*tls.Certificate

	listener net.Listener
}

func (s *WorkloadClusterListener) Host() string {
	return s.host
}

func (s *WorkloadClusterListener) Port() int {
	return s.port
}

func (s *WorkloadClusterListener) Address() string {
	return fmt.Sprintf("https://%s", s.HostPort())
}

func (s *WorkloadClusterListener) HostPort() string {
	return net.JoinHostPort(s.host, fmt.Sprintf("%d", s.port))
}

func (s *WorkloadClusterListener) RESTConfig() (*rest.Config, error) {
	kubeConfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"goofy": {
				Server:                   s.Address(),
				CertificateAuthorityData: certs.EncodeCertPEM(s.apiServerCaCertificate), // TODO: convert to PEM (store in double format
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"goofy": {
				Username:              "goofy",
				ClientCertificateData: certs.EncodeCertPEM(s.adminCertificate), // TODO: convert to PEM
				ClientKeyData:         certs.EncodePrivateKeyPEM(s.adminKey),   // TODO: convert to PEM
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"goofy": {
				Cluster:  "goofy",
				AuthInfo: "goofy",
			},
		},
		CurrentContext: "goofy",
	}

	b, err := clientcmd.Write(kubeConfig)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(b)
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (s *WorkloadClusterListener) GetClient() (client.Client, error) {
	restConfig, err := s.RESTConfig()
	if err != nil {
		return nil, err
	}

	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, err
	}

	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, httpClient)
	if err != nil {
		return nil, err
	}

	c, err := client.New(restConfig, client.Options{Scheme: s.scheme, Mapper: mapper})
	if err != nil {
		return nil, err
	}

	return c, err
}
