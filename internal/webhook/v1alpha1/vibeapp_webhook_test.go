/*
Copyright 2026.

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

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsexamplecomv1alpha1 "apps.example.com/vibe-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("VibeApp Webhook", func() {
	var (
		ctx       context.Context
		obj       *appsexamplecomv1alpha1.VibeApp
		oldObj    *appsexamplecomv1alpha1.VibeApp
		validator VibeAppCustomValidator
		defaulter VibeAppCustomDefaulter
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &appsexamplecomv1alpha1.VibeApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vibeapp",
				Namespace: "default",
			},
			Spec: appsexamplecomv1alpha1.VibeAppSpec{},
		}
		oldObj = &appsexamplecomv1alpha1.VibeApp{}
		validator = VibeAppCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = VibeAppCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When applying Defaulting Webhook", func() {
		It("Should add created-by label when not set", func() {
			Expect(obj.Labels).To(BeNil(), "Labels should be nil initially")

			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred(), "Default should not error")
			Expect(obj.Labels).NotTo(BeNil())
			Expect(obj.Labels["created-by"]).To(Equal("wang"))
		})

		It("Should override created-by label if already set", func() {
			obj.Labels = map[string]string{
				"created-by": "someone-else",
			}
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Labels["created-by"]).To(Equal("wang"))
		})

		It("Should set replicas to 2 when replicas is 0", func() {
			obj.Spec.Replicas = 0
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Replicas).To(Equal(int32(2)))
		})

		It("Should set replicas to 2 when replicas is 1", func() {
			obj.Spec.Replicas = 1
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Replicas).To(Equal(int32(2)))
		})

		It("Should not change replicas when already >= 2", func() {
			obj.Spec.Replicas = 5
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Replicas).To(Equal(int32(5)))
		})

		It("Should apply both label and replicas default together", func() {
			obj.Spec.Replicas = 1
			obj.Labels = map[string]string{}
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Replicas).To(Equal(int32(2)))
			Expect(obj.Labels["created-by"]).To(Equal("wang"))
		})
	})

	Context("When validating VibeApp creation", func() {
		validate := func(vibeapp *appsexamplecomv1alpha1.VibeApp) error {
			_, err := validator.ValidateCreate(ctx, vibeapp)
			return err
		}

		It("Should reject when image is missing", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.image is required"))
		})

		It("Should reject when port is 0", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            0,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.port must be a positive integer"))
		})

		It("Should reject when port is negative", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            -1,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.port must be a positive integer"))
		})

		It("Should reject when replicas < 2", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        1,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.replicas must be at least 2"))
		})

		It("Should reject when healthCheckPath is empty", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.healthCheckPath is required"))
		})

		It("Should reject when storagePath is empty", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "",
				Port:            80,
			}
			err := validate(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.storagePath is required"))
		})

		It("Should accept valid VibeApp", func() {
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validate(obj)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When validating VibeApp update", func() {
		validateUpdate := func(oldObj, newObj *appsexamplecomv1alpha1.VibeApp) error {
			_, err := validator.ValidateUpdate(ctx, oldObj, newObj)
			return err
		}

		It("Should reject update when image is removed", func() {
			oldObj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:latest",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validateUpdate(oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.image is required"))
		})

		It("Should accept valid update", func() {
			oldObj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:1.19",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			obj.Spec = appsexamplecomv1alpha1.VibeAppSpec{
				Image:           "nginx:1.20",
				Replicas:        2,
				HealthCheckPath: "/healthz",
				StoragePath:     "/data",
				Port:            80,
			}
			err := validateUpdate(oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
