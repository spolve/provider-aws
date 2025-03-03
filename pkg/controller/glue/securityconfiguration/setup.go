/*
Copyright 2021 The Crossplane Authors.
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

package securityconfiguration

import (
	"context"

	svcsdk "github.com/aws/aws-sdk-go/service/glue"
	ctrl "sigs.k8s.io/controller-runtime"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	svcapitypes "github.com/crossplane-contrib/provider-aws/apis/glue/v1alpha1"
	"github.com/crossplane-contrib/provider-aws/apis/v1alpha1"
	awsclients "github.com/crossplane-contrib/provider-aws/pkg/clients"
	"github.com/crossplane-contrib/provider-aws/pkg/features"
)

// SetupSecurityConfiguration adds a controller that reconciles SecurityConfiguration.
func SetupSecurityConfiguration(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(svcapitypes.SecurityConfigurationGroupKind)
	opts := []option{
		func(e *external) {
			e.postCreate = postCreate
			e.preDelete = preDelete
			e.preObserve = preObserve
			e.postObserve = postObserve
			e.preCreate = preCreate
		},
	}

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), v1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&svcapitypes.SecurityConfiguration{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(svcapitypes.SecurityConfigurationGroupVersionKind),
			managed.WithExternalConnecter(&connector{kube: mgr.GetClient(), opts: opts}),
			managed.WithPollInterval(o.PollInterval),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

func preDelete(_ context.Context, cr *svcapitypes.SecurityConfiguration, obj *svcsdk.DeleteSecurityConfigurationInput) (bool, error) {
	obj.Name = awsclients.String(meta.GetExternalName(cr))
	return false, nil
}

func preObserve(_ context.Context, cr *svcapitypes.SecurityConfiguration, obj *svcsdk.GetSecurityConfigurationInput) error {
	obj.Name = awsclients.String(meta.GetExternalName(cr))
	return nil
}

func postObserve(_ context.Context, cr *svcapitypes.SecurityConfiguration, obj *svcsdk.GetSecurityConfigurationOutput, obs managed.ExternalObservation, err error) (managed.ExternalObservation, error) {
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	cr.SetConditions(xpv1.Available())
	return obs, nil
}

func postCreate(_ context.Context, cr *svcapitypes.SecurityConfiguration, obj *svcsdk.CreateSecurityConfigurationOutput, _ managed.ExternalCreation, err error) (managed.ExternalCreation, error) {
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	meta.SetExternalName(cr, awsclients.StringValue(obj.Name))
	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func preCreate(_ context.Context, cr *svcapitypes.SecurityConfiguration, obj *svcsdk.CreateSecurityConfigurationInput) error {
	obj.Name = awsclients.String(meta.GetExternalName(cr))

	if cr.Spec.ForProvider.CustomEncryptionConfiguration != nil {
		obj.EncryptionConfiguration = &svcsdk.EncryptionConfiguration{
			CloudWatchEncryption: &svcsdk.CloudWatchEncryption{
				CloudWatchEncryptionMode: cr.Spec.ForProvider.CustomEncryptionConfiguration.CustomCloudWatchEncryption.CloudWatchEncryptionMode,
				KmsKeyArn:                cr.Spec.ForProvider.CustomEncryptionConfiguration.CustomCloudWatchEncryption.KMSKeyARN,
			},
			JobBookmarksEncryption: &svcsdk.JobBookmarksEncryption{
				JobBookmarksEncryptionMode: cr.Spec.ForProvider.CustomEncryptionConfiguration.CustomJobBookmarksEncryption.JobBookmarksEncryptionMode,
				KmsKeyArn:                  cr.Spec.ForProvider.CustomEncryptionConfiguration.CustomJobBookmarksEncryption.KMSKeyARN,
			},
		}
	}

	return nil
}
