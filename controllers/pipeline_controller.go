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
	"fmt"

	"github.com/go-logr/logr"
	ppsv1 "github.com/pachyderm/pipeline-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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

		pipeline.Status.State = "Running"

		if err := r.Status().Update(ctx, &pipeline); err != nil {
			log.Error(err, "unable to update Pipeline status")
			return ctrl.Result{}, err
		}

	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = ppsv1.GroupVersion.String()
)

func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		// ...make sure it's a CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "Pipeline" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ppsv1.Pipeline{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// newDeployment creates a new Deployment for a Foo resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Foo resource that 'owns' it.
func newDeployment(pipeline *ppsv1.Pipeline) *appsv1.Deployment {
	workerImage := "pachyderm/worker:local"
	pachImage := "pachyderm/pachd:local"
	//volumeMounts := ""
	userImage := pipeline.Spec.Transform.Image

	//Need to know storage backend (ie GCS Bucket ) +  Secret

	//What about standby pipelines? - Need a way to get signal from master to spin up again

	//Will controller look at pach etcd for state?

	//Will controller connect to PFS (ie to save pipeline spec)

	//Worker Env
	//TODO Add transform.Env

	userEnvVars := []corev1.EnvVar{{
		Name:  "STORAGE_BACKEND",
		Value: "LOCAL",
	}, {
		Name:  "PACH_ROOT",
		Value: "/pach",
	}, {
		Name:  "PEER_PORT",
		Value: "653",
	}, {
		Name:  "PPS_PIPELINE_NAME",
		Value: pipeline.GetObjectMeta().GetName(),
	}, {
		Name:  "PPS_WORKER_GRPC_PORT",
		Value: "80",
	}, {
		Name: "PPS_WORKER_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "status.podIP",
			},
		},
	}, {
		Name: "PPS_POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		},
	}, {
		Name:  "PPS_SPEC_COMMIT",
		Value: pipeline.Spec.SpecCommitId,
	},
	}

	sidecarEnvVars := []corev1.EnvVar{{
		Name:  "PACH_ROOT",
		Value: "/pach",
	}, {
		Name:  "PACH_NAMESPACE",
		Value: "default",
	}, {
		Name:  "BLOCK_CACHE_BYTES",
		Value: "64M",
	}, {
		Name:  "PFS_CACHE_SIZE",
		Value: "16",
	}, {
		Name:  "STORAGE_BACKEND",
		Value: "Local",
	}, {
		Name:  "PPS_WOKER_GRPC_PORT",
		Value: "80",
	}, {
		Name:  "PORT",
		Value: "650",
	}, {
		Name:  "HTTP_PORT",
		Value: "652",
	}, {
		Name:  "PEER_PORT",
		Value: "653",
	}, {
		Name: "PACHD_POD_NAME",
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		},
	},
	}

	initVolumeMounts := []corev1.VolumeMount{{
		Name:      "pach-bin",
		MountPath: "/pach-bin",
	}, {
		Name:      "pachyderm-worker",
		MountPath: "/pfs",
	}}

	storageVolumeMounts := []corev1.VolumeMount{{
		Name:      "pach-disk",
		MountPath: "/pach",
	}} //Missing storage secret

	userVolumeMounts := []corev1.VolumeMount{{
		Name:      "pach-bin",
		MountPath: "/pach-bin",
	}, {
		Name:      "pachyderm-worker",
		MountPath: "/pfs",
	}, {
		Name:      "pach-disk",
		MountPath: "/pach",
	}} //Missing storage secret and docker

	labels := map[string]string{
		"app":        "pachd",
		"controller": pipeline.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pipeline-%s-v1", pipeline.Name),
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
							//Command: []string{"/pach/worker.sh"},
							//ImagePullPolicy: corev1.PullPolicy(pullPolicy),
							VolumeMounts: initVolumeMounts, //options.volumeMounts,
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
							/*
								Command: []string{"/pach-bin/dlv"},
								Args: []string{
									"exec", "/pach-bin/worker",
									"--listen=:2345",
									"--headless=true",
									//"--log=true",
									//"--log-output=debugger,debuglineerr,gdbwire,lldbout,rpc",
									"--accept-multiclient",
									"--api-version=2",
								},
							*/
							//ImagePullPolicy: v1.PullPolicy(pullPolicy),
							Env: userEnvVars,
							/*Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuZeroQuantity,
									v1.ResourceMemory: memDefaultQuantity,
								},
							},*/
							VolumeMounts: userVolumeMounts,
						},
						{
							Name:  "storage", //client.PPSWorkerSidecarContainerName,
							Image: pachImage, //a.workerSidecarImage,
							//Command: []string{"/app/pachd", "--mode", "sidecar"},
							Command: []string{"/pachd", "--mode", "sidecar"},
							//ImagePullPolicy: v1.PullPolicy(pullPolicy),
							Env:          sidecarEnvVars,
							VolumeMounts: storageVolumeMounts,
							/*Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuZeroQuantity,
									v1.ResourceMemory: memSidecarQuantity,
								},
							},*/
							//Ports: sidecarPorts,
						},
					},
					ServiceAccountName: "pachyderm-worker",
					//RestartPolicy: "Always",
					Volumes: []corev1.Volume{{
						Name: "pach-disk",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/pachyderm/pachd",
							},
						},
					}, {
						Name: "pachyderm-worker",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}, {
						Name: "pach-bin",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}},
					//ImagePullSecrets:              options.imagePullSecrets,
					//TerminationGracePeriodSeconds: &zeroVal,
					//SecurityContext:               securityContext,
				},
			},
		},
	}
}
