/*
Copyright 2021 Wim Henderickx.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/mwitkow/grpc-proxy/proxy"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	fscv1 "github.com/fsc-demo-wim/fsc-proxy/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// DefaultTokenTTL is a dummy time
	DefaultTokenTTL = 10 * time.Minute
)

// FscProxyReconciler reconciles a FscProxy object
type FscProxyReconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Ctx     context.Context
	Proxies map[string]*Proxy
}

// Proxy is a struct that contains the proxy informaation
type Proxy struct {
	frontEndAddress   string
	frontEndCertFile  string
	frontEndKeyFile   string
	backEndAddress    string
	backEndCertFile   string
	backEndKeyFile    string
	backEndServerName string
	backEndFilter     string
	server            *grpc.Server
}

// +kubebuilder:rbac:groups=fsc.henderiw.be,resources=fscproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fsc.henderiw.be,resources=fscproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fsc.henderiw.be,resources=fscproxies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FscProxy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *FscProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var port string
	r.Log.WithValues(req.Name, req.Namespace).Info("reconciling FscProxy")

	p := &fscv1.FscProxy{}
	if err := r.Client.Get(ctx, req.NamespacedName, p); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		r.Log.WithValues(req.Name, req.Namespace).Error(err, "Failed to get fscProxy config")
		return ctrl.Result{}, err
	}

	if p.Status.FreeFrontEndPortRange == nil {
		p.Status.FreeFrontEndPortRange = &p.Spec.FrontEndPortRange
	}
	freePorts := *p.Status.FreeFrontEndPortRange

	// get network node list
	selectors := []client.ListOption{
		client.MatchingLabels{},
	}
	nns := &fscv1.NetworkNodeList{}
	if err := r.Client.List(r.Ctx, nns, selectors...); err != nil {
		r.Log.WithValues(req.Name, req.Namespace).Error(err, "Failed to get NetworkNode ")
		return ctrl.Result{}, err
	}

	// check the status of the network node, if ready start/allocate port for the proxy if not already started
	for _, nn := range nns.Items {
		// Check if proxy is started for this network node, if not start it
		if nn.GetDiscoveryStatus() == "Ready" {
			r.Log.WithValues(nn.GetName(), nn.GetNamespace(), "status", nn.GetDiscoveryStatus()).Info("Check if proxy is started for this network node, if not start it...")
			if p.Status.Targets == nil {
				p.Status.Targets = make(map[string]*fscv1.Target)
			}

			// proxy target was not started
			if _, ok := p.Status.Targets[nn.GetName()]; !ok {
				port, freePorts, err = allocateProxyPort(freePorts)
				if err != nil {
					return ctrl.Result{}, errors.Wrap(err,
						fmt.Sprintf("failed to allocate port"))
				}

				r.Proxies[nn.GetName()] = &Proxy{
					frontEndAddress:   "127.0.0.1:" + port,
					frontEndCertFile:  "",
					frontEndKeyFile:   "",
					backEndAddress:    nn.Spec.Target.Address,
					backEndCertFile:   "",
					backEndKeyFile:    "",
					backEndFilter:     "/",
					backEndServerName: nn.GetName(),
				}

				s, err := getServer(r.Proxies[nn.GetName()])
				if err != nil {
					return ctrl.Result{}, errors.Wrap(err,
						fmt.Sprintf("failed to get server"))
				}

				r.Proxies[nn.GetName()].server = s

				go func() {
					r.startProxy(r.Proxies[nn.GetName()])
				}()

				r.Log.WithValues(req.Name, req.Namespace, "frontEndPort", "127.0.0.1:"+port).Info("proxy server started ...")

				p.Status.FreeFrontEndPortRange = &freePorts
				p.Status.Targets[nn.GetName()] = &fscv1.Target{
					FrontEndAddress: "localhost:" + port,
					BackEndAddress:  nn.Spec.Target.Address,
				}
				if err = r.saveProxyStatus(ctx, p); err != nil {
					return ctrl.Result{}, errors.Wrap(err,
						fmt.Sprintf("failed to save proxy status"))
				}
			}
			// do nothing, since the proxy was already started for this networkNode
		}
		// Check if proxy was started for this network node, if not stop/deallocate it
		if nn.GetDiscoveryStatus() != "Ready" {
			r.Log.WithValues(nn.GetName(), nn.GetNamespace(), "status", nn.GetDiscoveryStatus()).Info("Check if proxy was started for this network node, if so stop it...")
			// proxy target was started
			if _, ok := p.Status.Targets[nn.GetName()]; ok {
				port := strings.Split(p.Status.Targets[nn.GetName()].FrontEndAddress, ":")
				freePorts, err = deallocateProxyPort(port[1], freePorts)
				if err != nil {
					return ctrl.Result{}, errors.Wrap(err,
						fmt.Sprintf("failed to deallocate port"))
				}

				r.stopProxy(r.Proxies[nn.GetName()])

				p.Status.FreeFrontEndPortRange = &freePorts
				delete(p.Status.Targets, nn.GetName())
				if err = r.saveProxyStatus(ctx, p); err != nil {
					return ctrl.Result{}, errors.Wrap(err,
						fmt.Sprintf("failed to save proxy status"))
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FscProxyReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, option controller.Options) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&fscv1.FscProxy{}).
		WithOptions(option).
		Watches(
			&source.Kind{Type: &fscv1.NetworkNode{}},
			handler.EnqueueRequestsFromMapFunc(r.NetworkNodeMapFunc),
		)

	_, err := b.Build(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up with a controller manager")
	}
	return nil
}

// NetworkNodeMapFunc is a handler.ToRequestsFunc to be used to enqeue
// request for reconciliation of FscProxy.
func (r *FscProxyReconciler) NetworkNodeMapFunc(o client.Object) []ctrl.Request {
	result := []ctrl.Request{}

	nn, ok := o.(*fscv1.NetworkNode)
	if !ok {
		panic(fmt.Sprintf("Expected a NetworkNode but got a %T", o))
	}
	r.Log.WithValues(nn.GetName(), nn.GetNamespace()).Info("NetworkNode MapFunction")

	name := client.ObjectKey{
		Namespace: nn.GetNamespace(),
		Name:      nn.GetName(),
	}

	result = append(result, ctrl.Request{NamespacedName: name})

	return result
}

// saveProxyStatus
func (r *FscProxyReconciler) saveProxyStatus(ctx context.Context, p *fscv1.FscProxy) error {
	t := metav1.Now()
	p.Status.DeepCopy()
	p.Status.LastUpdated = &t

	if err := r.Client.Status().Update(ctx, p); err != nil {
		r.Log.WithValues(p.Name, p.Namespace).Error(err, "Failed to update proxy ")
		return err
	}
	return nil

	//return r.Client.Status().Update(ctx, nn)
}

func allocateProxyPort(free string) (string, string, error) {
	var err error
	var port string
	freePort := strings.Split(free, ",")
	// allocate from individual port freelist "50000,50002:50100"
	if len(freePort) > 1 {
		port = freePort[0]
		free = strings.TrimLeft(free, freePort[0])
		free = strings.TrimLeft(free, ",")
		return port, free, err
	}
	// allocate from block freelist "50000:50100"
	freePort = strings.Split(free, ":")
	if len(freePort) > 1 {
		startPort, err := strconv.Atoi(freePort[0])
		if err != nil {
			return "", "", err
		}
		stopPort, err := strconv.Atoi(freePort[1])
		if err != nil {
			return "", "", err
		}
		if startPort+1 < stopPort {
			free = strconv.Itoa(startPort+1) + ":" + freePort[1]
			return strconv.Itoa(startPort), free, err
		}
		return strconv.Itoa(startPort), strconv.Itoa(stopPort), err
	}
	// allocate the last element in the free block
	return free, "", err
}

func deallocateProxyPort(port, free string) (string, error) {
	var err error
	freePort := strings.Split(free, ",")
	// free port list was split, we dont optimize it, "50000,50002:50100"
	// TODO optimize the split
	if len(freePort) > 1 {
		return port + "," + free, nil
	}
	// free port list was is a block, "50000:50100"
	freePort = strings.Split(free, ":")
	if len(freePort) > 1 {
		startPort, err := strconv.Atoi(freePort[0])
		if err != nil {
			return "", err
		}
		deallocPort, err := strconv.Atoi(port)
		if err != nil {
			return "", err
		}
		if deallocPort == startPort-1 {
			return port + ":" + freePort[1], nil
		}
		return port + "," + free, nil

	}
	// freelist was empty
	return port, err
}

func (r *FscProxyReconciler) startProxy(p *Proxy) error {
	l, err := net.Listen("tcp", p.frontEndAddress)
	if err != nil {
		return err
	}

	r.Log.WithValues("name", p.backEndServerName).Info("started server...")

	if err := p.server.Serve(l); err != nil {
		return err
	}

	return nil
}

func (r *FscProxyReconciler) stopProxy(p *Proxy) error {
	p.server.GracefulStop()

	r.Log.WithValues("name", p.backEndServerName).Info("stopped server...")
	return nil
}

func getServer(p *Proxy) (*grpc.Server, error) {
	var opts []grpc.ServerOption

	opts = append(opts, grpc.CustomCodec(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(getDirector(p))))

	if p.frontEndCertFile != "" && p.frontEndKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(p.frontEndCertFile, p.frontEndKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate credentials %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	return grpc.NewServer(opts...), nil
}

func getDirector(p *Proxy) func(context.Context, string) (context.Context, *grpc.ClientConn, error) {

	return func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {

		if strings.HasPrefix(fullMethodName, p.backEndFilter) {
			fmt.Printf("Found: %s > %s \n", fullMethodName, p.backEndAddress)
			if p.backEndCertFile == "" {
				g, err := grpc.DialContext(ctx, p.backEndAddress, grpc.WithCodec(proxy.Codec()),
					grpc.WithInsecure())
				if err != nil {
					return nil, nil, err
				}
				return ctx, g, nil
			}
			creds := GetCredentials(p)
			if creds != nil {
				g, err := grpc.DialContext(ctx, p.backEndAddress, grpc.WithCodec(proxy.Codec()),
					grpc.WithTransportCredentials(creds))
				if err != nil {
					return nil, nil, err
				}
				return ctx, g, nil
			}
			return nil, nil, grpc.Errorf(codes.FailedPrecondition, "Backend TLS is not configured properly in grpc-proxy")
		}
		return nil, nil, grpc.Errorf(codes.Unimplemented, "Unknown method")
	}
}

// GetCredentials function
func GetCredentials(p *Proxy) credentials.TransportCredentials {
	creds, err := credentials.NewClientTLSFromFile(p.backEndCertFile, p.backEndServerName)
	if err != nil {
		grpclog.Fatalf("Failed to create TLS credentials %v", err)
		return nil
	}
	return creds
}
