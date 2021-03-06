/*
Copyright 2022.

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
	"github.com/git-czy/cluster-api-metalnode/api/v1beta1"
	"github.com/git-czy/cluster-api-metalnode/pkg/kubeadm/cloudinit"
	"github.com/git-czy/cluster-api-metalnode/pkg/remote"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	util "github.com/git-czy/cluster-api-metalnode/utils"
	"github.com/git-czy/cluster-api-metalnode/utils/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	INITIALIZING v1beta1.InitializationState = "INITIALIZING"
	CHECKING     v1beta1.InitializationState = "CHECKING"
	FAIL         v1beta1.InitializationState = "FAIL"
	SUCCESS      v1beta1.InitializationState = "SUCCESS"
	READY        bool                        = true
)

// MetalNodeReconciler reconciles a MetalNode object
type MetalNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=bocloud.io,resources=metalnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bocloud.io,resources=metalnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bocloud.io,resources=metalnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MetalNode object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *MetalNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	metalNode := &v1beta1.MetalNode{}
	if err := r.Get(ctx, req.NamespacedName, metalNode); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.WithError(err).Error("unable to fetch MetalNode")
		return ctrl.Result{}, err
	}

	if err := metalNode.Spec.NodeEndPoint.Validate(); err != nil {
		log.WithError(err).Errorln("Invalid metal node endpoint host")
		return ctrl.Result{}, err
	}
	l := log.With("metalnode", metalNode.Name).With("host", metalNode.Spec.NodeEndPoint.Host)

	// always update the status of the metal node,when leave reconcile
	defer func() {
		if metalNode.Status.InitializationState == SUCCESS {
			metalNode.Status.Ready = READY
		}
		if err := r.Status().Update(ctx, metalNode); err != nil {
			l.WithError(err).Errorln("failed to update metal node status")
		}
	}()

	// if metal node is INITIALIZING, do nothing
	if metalNode.Status.InitializationState == INITIALIZING {
		return ctrl.Result{}, nil
	}

	// if metal node is CHECKING, do nothing
	if metalNode.Status.InitializationState == CHECKING {
		return ctrl.Result{}, nil
	}

	if metalNode.Status.InitializationState == "" {
		metalNode.Status.InitializationState = INITIALIZING
		metalNode.Status.Bootstrapped = false
		metalNode.Status.Ready = false
		if err := r.Status().Update(ctx, metalNode); err != nil {
			l.WithError(err).Errorln("failed to update metal node status")
			return ctrl.Result{}, err
		}

		if err := r.initMetal(ctx, metalNode); err != nil {
			l.WithError(err).Errorln("failed to initialize metal node")
			return ctrl.Result{}, err
		}

		metalNode.Status.InitializationState = CHECKING
		if err := r.Status().Update(ctx, metalNode); err != nil {
			l.WithError(err).Errorln("failed to update metal node status")
			return ctrl.Result{}, err
		}
		// if metal node InitializationFailureReason is not empty, maybe means the initialization failed
		// so need to check the metal node is initialized or not(check docker kubelet kubeadm)
		if err := r.checkMetalNodeInitialized(ctx, metalNode); err != nil {
			metalNode.Status.InitializationState = FAIL
			l.WithError(err).Errorln("failed to initialize metal node")
			return ctrl.Result{}, errors.New("metal node initialization failed")
		}

		l.Info("initialized metal node successfully")

		metalNode.Status.InitializationState = SUCCESS
		return ctrl.Result{}, nil
	}

	if metalNode.Status.DataSecretName != "" && !metalNode.Status.Bootstrapped {
		err := r.bootstrapMetalNode(ctx, metalNode)
		if err != nil {
			l.WithError(err).Errorln("failed to bootstrap metal node")
			return ctrl.Result{}, err
		}
		if err = r.checkMetalNodeBootstrap(ctx, metalNode); err != nil {
			l.Error("failed to check metal node bootstrap")
			return ctrl.Result{}, err
		}
		metalNode.Status.Bootstrapped = true
		l.Infoln("bootstrapped metal node successfully")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetalNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.MetalNode{}).
		Complete(r)
}

// initMetal initializes the metal node
func (r *MetalNodeReconciler) initMetal(ctx context.Context, metalNode *v1beta1.MetalNode) error {
	host := metalNodeToHost(metalNode)
	cmd := remote.Command{
		Cmds: []string{
			"sudo chmod +x /tmp/init_k8s_env.sh",
			"sudo sed -i 's/\\r//g' /tmp/init_k8s_env.sh",
			"sudo /bin/bash /tmp/init_k8s_env.sh",
			"sudo hostnamectl set-hostname " + host[0].Address,
		},
		FileUp: []remote.File{
			{Src: "script/init_k8s_env.sh", Dst: "/tmp"},
		},
	}

	if metalNode.Spec.InitializationCmd != nil {
		cmd.Cmds = metalNode.Spec.InitializationCmd
	}

	errs := remote.Run(host, cmd)
	if len(errs[metalNode.Spec.NodeEndPoint.Host]) != 0 {
		metalNode.Status.InitializationFailureReason = errs[metalNode.Spec.NodeEndPoint.Host]
		if err := r.Status().Update(ctx, metalNode); err != nil {
			return err
		}
	}
	return nil
}

// check metal node is already initialized
func (r *MetalNodeReconciler) checkMetalNodeInitialized(ctx context.Context, metalNode *v1beta1.MetalNode) error {
	host := metalNodeToHost(metalNode)
	cmd := remote.Command{
		Cmds: []string{
			"sudo docker version",
			"kubelet --version",
			"kubectl version",
		},
	}

	//if metalNode.Spec.InitializationCmd != nil {
	//	cmd = *metalNode.Spec.InitializationCmd
	//}

	errs := remote.Run(host, cmd)

	ignoreErrs := []string{
		fmt.Sprintf("The connection to the server %s:6443 was refused - did you specify the right host or port?", host[0].Address),
		fmt.Sprintf("The connection to the server %s:8080 was refused - did you specify the right host or port?", host[0].Address),
		"The connection to the server localhost:6443 was refused - did you specify the right host or port?",
		"The connection to the server localhost:8080 was refused - did you specify the right host or port?",
	}

	// it's possible to get one err when run kubectl version,but we don't care about it, because not bootstrap yet
	checkErrs := util.SliceExcludeSlice(errs[metalNode.Spec.NodeEndPoint.Host], ignoreErrs)

	if len(checkErrs) != 0 {
		metalNode.Status.CheckFailureReason = checkErrs
		if err := r.Status().Update(ctx, metalNode); err != nil {
			return err
		}
		return errors.New("metal node is initialized failed")
	}
	return nil
}

// bootstrapMetalNode bootstrap the metal node with bootstrap data
func (r *MetalNodeReconciler) bootstrapMetalNode(ctx context.Context, metalNode *v1beta1.MetalNode) error {
	host := metalNodeToHost(metalNode)
	cmd, err := r.getBootstrapDataToCmds(ctx, metalNode)
	if err != nil {
		return err
	}

	errs := remote.Run(host, *cmd)
	if len(errs[metalNode.Spec.NodeEndPoint.Host]) != 0 {
		metalNode.Status.BootstrapFailureReason = errs[metalNode.Spec.NodeEndPoint.Host]
		if err := r.Status().Update(ctx, metalNode); err != nil {
			return err
		}
	}
	return nil
}

func (r *MetalNodeReconciler) checkMetalNodeBootstrap(ctx context.Context, metalNode *v1beta1.MetalNode) error {
	host := metalNodeToHost(metalNode)
	cmd := remote.Command{
		Cmds: remote.Commands{
			"sudo cat /run/cluster-api/bootstrap-success.complete",
		},
	}
	errs := remote.Run(host, cmd)

	if len(errs[metalNode.Spec.NodeEndPoint.Host]) != 0 {
		return errors.New("metal node bootstrap failed")
	}
	return nil
}

//getBootstrapDataToCmds get bootstrap data from secret and converts to remote.Command
func (r *MetalNodeReconciler) getBootstrapDataToCmds(ctx context.Context, metalNode *v1beta1.MetalNode) (*remote.Command, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: metalNode.Status.DataSecretName, Namespace: metalNode.Namespace}, secret); err != nil {
		return nil, err
	}
	config, ok := secret.Data["value"]
	if !ok {
		return nil, errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	format, ok := secret.Data["format"]
	if !ok {
		return nil, errors.New("error retrieving bootstrap data: secret format key is missing")
	}

	parser := cloudinit.NewBootstrapDataParser()

	cmd, err := parser.Parse(config, format)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

func metalNodeToHost(metalNode *v1beta1.MetalNode) []remote.Host {
	return []remote.Host{
		{
			User:     metalNode.Spec.NodeEndPoint.SSHAuth.User,
			Password: metalNode.Spec.NodeEndPoint.SSHAuth.Password,
			Address:  metalNode.Spec.NodeEndPoint.Host,
			Port:     metalNode.Spec.NodeEndPoint.SSHAuth.Port,
			SSHKey:   metalNode.Spec.NodeEndPoint.SSHAuth.SSHKey,
		},
	}
}
