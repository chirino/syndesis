package internal

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/syndesisio/syndesis/install/operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Options struct {
	KubeConfig string
	Namespace  string

	Context context.Context
	Command *cobra.Command
	Client  *client.Client
}

func (o *Options) GetClientConfig() *rest.Config {
	c, err := config.GetConfig()
	util.ExitOnError(err)
	return c
}

func (o *Options) GetClient() (c client.Client, err error) {
	if o.Client == nil {
		cl, err := client.New(o.GetClientConfig(), client.Options{})
		if err != nil {
			return nil, err
		}
		o.Client = &cl
	}
	return *o.Client, nil
}

func (o *Options) NewDynamicClient() (c dynamic.Interface, err error) {
	return dynamic.NewForConfig(o.GetClientConfig())
}

func (o *Options) NewApiClient() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(o.GetClientConfig())
}

type ExecOptions struct {
	remotecommand.StreamOptions
	Pod       string
	Container string
	Command   []string
}

func (o *Options) Exec(request ExecOptions) error {
	config := o.GetClientConfig()
	api, err := o.NewApiClient()
	if err != nil {
		return err
	}
	req := api.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(request.Pod).
		Namespace(o.Namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: request.Container,
			Command:   request.Command,
			Stdout:    request.Stdout != nil,
			Stderr:    request.Stderr != nil,
			Stdin:     request.Stdin != nil,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	return exec.Stream(request.StreamOptions)
}
