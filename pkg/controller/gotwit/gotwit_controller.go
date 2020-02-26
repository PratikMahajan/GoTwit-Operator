package gotwit

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	twtv1alpha1 "github.com/pratikmahajan/GoTwit-Operator/pkg/apis/twt/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_gotwit")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new GoTwit Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGoTwit{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gotwit-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GoTwit
	err = c.Watch(&source.Kind{Type: &twtv1alpha1.GoTwit{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner GoTwit
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &twtv1alpha1.GoTwit{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGoTwit implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGoTwit{}

// ReconcileGoTwit reconciles a GoTwit object
type ReconcileGoTwit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a GoTwit object and makes changes based on the state read
// and what is in the GoTwit.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGoTwit) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GoTwit")

	// Fetch the GoTwit instance
	instance := &twtv1alpha1.GoTwit{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// create a new deployment
	deployment := r.newGoTwtDeployment(instance)

	// Check if the pod already exists
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace:deployment.Namespace, Name:deployment.Name} , found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.newGoTwtDeployment(instance)
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	// create a new service
	service := r.newServiceForGoTwt(instance)

	// check if service already exists
	serviceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace:service.Namespace}, serviceFound)
	if err != nil && errors.IsNotFound(err){
		// define a new service
		ser := r.newServiceForGoTwt(instance)
		reqLogger.Info("Creating a new Service", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
		err = r.client.Create(context.TODO(), ser)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
			return reconcile.Result{}, err
		}
		// Service created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service")
		return reconcile.Result{}, err
	}

	size := instance.Spec.Size
	if *found.Spec.Replicas != size{
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the Memcached status with the pod names
	// List the pods for this memcached's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labelsForInstance(instance.Name)),
	}
	if err = r.client.List(context.TODO(), podList, listOpts...); err != nil {
		reqLogger.Error(err, "Failed to list pods", "Instance.Namespace", instance.Namespace, "Instance.Name", instance.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, instance.Status.Nodes) {
		instance.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update instance status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil


}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForInstance(name string) map[string]string {
	return map[string]string{"app": "gotwt", "instance_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}


// Write the YAML file in a go format to deploy pods
func (r *ReconcileGoTwit) newGoTwtDeployment(gt *twtv1alpha1.GoTwit )  *appsv1.Deployment {
	log.Info(fmt.Sprintf("Initiating deployment"))
	ls := labelsForInstance(gt.Name)
	replicas := gt.Spec.Size

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gt.Name,
			Namespace: gt.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "quay.io/pratikmahajan/go-app-twitter:staging-latest",
						Name:    "gotwt",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5000,
							Name:         "gotwt",
						}},
						Env: []corev1.EnvVar{{
							Name: "APP_HTTP_ADDR",
							Value: ":5000",
						}, {
							Name: "APP_ACCESSTOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									Key: "APP_ACCESSTOKEN",
									LocalObjectReference : corev1.LocalObjectReference{Name:"go-twitter-secret"},
								},
							},
						},{
							Name: "APP_ACCESSTOKENSECRET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									Key: "APP_ACCESSTOKENSECRET",
									LocalObjectReference : corev1.LocalObjectReference{Name:"go-twitter-secret"},
								},
							},
						},{
							Name: "APP_APIKEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									Key: "APP_APIKEY",
									LocalObjectReference : corev1.LocalObjectReference{Name:"go-twitter-secret"},
								},
							},
						},{
							Name: "APP_APISECRETKEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									Key: "APP_APISECRETKEY",
									LocalObjectReference : corev1.LocalObjectReference{Name:"go-twitter-secret"},
								},
							},
						},
						}},
					}},
				},
			},
		}
	// Set gotwt instance as the owner and controller of deployment
	err := controllerutil.SetControllerReference(gt, deployment, r.scheme)
	if err != nil {
		log.Error(err, fmt.Sprintf( "error setting deployment as owner and controller"))
	}
	return deployment

}

// Returns a new service
func (r *ReconcileGoTwit) newServiceForGoTwt(gt *twtv1alpha1.GoTwit) *corev1.Service {
	log.Info(fmt.Sprintf("Initiating deployment"))
	ls := labelsForInstance(gt.Name)

	Service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-" + gt.Name,
			Namespace: gt.Namespace,
			Labels: ls,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
					TargetPort: intstr.FromInt(5000),
				},
			},
		},
	}

	// set gotwt instance as owner and controller of service
	err := controllerutil.SetControllerReference(gt, Service, r.scheme)
	if err != nil {
		log.Error(err, fmt.Sprintf( "error setting service as owner and controller"))
	}
	return Service
}