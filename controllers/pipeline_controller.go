/*


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

	"github.com/go-logr/logr"
	ppsv1 "github.com/pachyderm/pipeline-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// PipelineReconciler reconciles a Pipeline object
type PipelineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pps.pachyderm.io,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pps.pachyderm.io,resources=pipelines/status,verbs=get;update;patch

func (r *PipelineReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("pipeline", req.NamespacedName)

	// your logic here
	var pipeline ppsv1.Pipeline
	if err := r.Get(ctx, req.NamespacedName, &pipeline); err != nil {
		//log.Info(err, "unable to fetch Pipeline")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pipelineDeployment := newDeployment(&pipeline)
	err := r.Get(ctx, types.NamespacedName{Name: pipelineDeployment.Name, Namespace: req.NamespacedName.Namespace}, &appsv1.Deployment{})
	if err != nil && errors.IsNotFound(err) {

		if err := controllerutil.SetControllerReference(&pipeline, pipelineDeployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Creating Deployment: ", pipelineDeployment.Namespace, pipelineDeployment.Name)

		err = r.Create(ctx, pipelineDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ppsv1.Pipeline{}).
		Complete(r)
}

// newDeployment creates a new Deployment for a Foo resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Foo resource that 'owns' it.
func newDeployment(pipeline *ppsv1.Pipeline) *appsv1.Deployment {
	workerImage := "pachyderm/worker:1.10.1"
	//volumeMounts := ""
	userImage := "pachyderm/worker:1.10.1"

	//Need to know storage backend (ie GCS Bucket ) +  Secret

	//What about standby pipelines? - Need a way to get signal from master to spin up again

	//Will controller look at pach etcd for state?

	//Will controller connect to PFS (ie to save pipeline spec)

	//Worker Env
	//TODO Add transform.Env

	labels := map[string]string{
		"app":        "pachd",
		"controller": pipeline.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipeline.Name,
			Namespace: pipeline.Namespace,
			/*OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pipeline, ppsv1.SchemeGroupVersion.WithKind("Pipeline")),
			},*/
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: 1,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "init",
							Image:   workerImage,
							Command: []string{"/app/init"},
							//ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							//VolumeMounts: volumeMounts, //options.volumeMounts,
							/*Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cpuZeroQuantity,
									corev1.ResourceMemory: memDefaultQuantity,
								},
							},*/
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "user",    //client.PPSWorkerUserContainerName,
							Image:   userImage, //options.userImage,
							Command: []string{"/pach-bin/worker"},
							//ImagePullPolicy: v1.PullPolicy(pullPolicy),
							/*Env: []v1.EnvVar{{
								Name:  client.PPSPipelineNameEnv,
								Value: pipelineInfo.Pipeline.Name,
							}},*/
							/*Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuZeroQuantity,
									v1.ResourceMemory: memDefaultQuantity,
								},
							},*/
							//VolumeMounts: userVolumeMounts,
						},
						{
							Name:    "storage",   //client.PPSWorkerSidecarContainerName,
							Image:   workerImage, //a.workerSidecarImage,
							Command: []string{"/app/pachd", "--mode", "sidecar"},
							//ImagePullPolicy: v1.PullPolicy(pullPolicy),
							//Env:          sidecarEnv,
							//VolumeMounts: sidecarVolumeMounts,
							/*Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuZeroQuantity,
									v1.ResourceMemory: memSidecarQuantity,
								},
							},*/
							//Ports: sidecarPorts,
						},
					},
					//ServiceAccountName: assets.WorkerSAName,
					RestartPolicy: "Always",
					//Volumes:            options.volumes,
					//ImagePullSecrets:              options.imagePullSecrets,
					//TerminationGracePeriodSeconds: &zeroVal,
					//SecurityContext:               securityContext,
				},
			},
		},
	}
}
