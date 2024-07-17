/*
Copyright 2024.

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

package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var PodAnnotatorlog = logf.Log.WithName("PodAnnotator-resource")

// +kubebuilder:webhook:path=/validate--v1-pod,mutating=false,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kb.io
type PodValidator struct{}
type PodAnnotator struct{}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PodValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodAnnotator{}).
		WithValidator(&PodValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-l2sm-l2sm-k8s-local-v1-PodAnnotator,mutating=true,failurePolicy=fail,sideEffects=None,groups=l2sm.l2sm.k8s.local,resources=PodAnnotators,verbs=create;update,versions=v1,name=mPodAnnotator.kb.io,admissionReviewVersions=v1

// var _ webhook.Defaulter = &PodValidator{}

func (v *PodValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod but got a %T", obj)
	}

	log.Info("Validating Pod")
	key := "example-mutating-admission-webhook"
	anno, found := pod.Annotations[key]
	if !found {
		return nil, fmt.Errorf("missing annotation %s", key)
	}
	if anno != "foo" {
		return nil, fmt.Errorf("annotation %s did not have value %q", key, "foo")
	}

	return nil, nil
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PodAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	// PodAnnotatorlog.Info("default", "name", r.Name)

	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations["example-mutating-admission-webhook"] = "foo"
	log.Info("Annotated Pod")

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-l2sm-l2sm-k8s-local-v1-PodAnnotator,mutating=false,failurePolicy=fail,sideEffects=None,groups=l2sm.l2sm.k8s.local,resources=PodAnnotators,verbs=create;update,versions=v1,name=vPodAnnotator.kb.io,admissionReviewVersions=v1

// var _ webhook.Validator = &PodValidator{}

func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *PodValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *PodValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

