/*
Copyright 2017 The Vault Initializer Authors.

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
package fake

import (
	v1alpha1 "github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVaultMaps implements VaultMapInterface
type FakeVaultMaps struct {
	Fake *FakeVaultinitV1alpha1
	ns   string
}

var vaultmapsResource = schema.GroupVersionResource{Group: "vaultinit.k8s.io", Version: "v1alpha1", Resource: "vaultmaps"}

var vaultmapsKind = schema.GroupVersionKind{Group: "vaultinit.k8s.io", Version: "v1alpha1", Kind: "VaultMap"}

// Get takes name of the vaultMap, and returns the corresponding vaultMap object, and an error if there is any.
func (c *FakeVaultMaps) Get(name string, options v1.GetOptions) (result *v1alpha1.VaultMap, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(vaultmapsResource, c.ns, name), &v1alpha1.VaultMap{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultMap), err
}

// List takes label and field selectors, and returns the list of VaultMaps that match those selectors.
func (c *FakeVaultMaps) List(opts v1.ListOptions) (result *v1alpha1.VaultMapList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(vaultmapsResource, vaultmapsKind, c.ns, opts), &v1alpha1.VaultMapList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VaultMapList{}
	for _, item := range obj.(*v1alpha1.VaultMapList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested vaultMaps.
func (c *FakeVaultMaps) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(vaultmapsResource, c.ns, opts))

}

// Create takes the representation of a vaultMap and creates it.  Returns the server's representation of the vaultMap, and an error, if there is any.
func (c *FakeVaultMaps) Create(vaultMap *v1alpha1.VaultMap) (result *v1alpha1.VaultMap, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(vaultmapsResource, c.ns, vaultMap), &v1alpha1.VaultMap{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultMap), err
}

// Update takes the representation of a vaultMap and updates it. Returns the server's representation of the vaultMap, and an error, if there is any.
func (c *FakeVaultMaps) Update(vaultMap *v1alpha1.VaultMap) (result *v1alpha1.VaultMap, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(vaultmapsResource, c.ns, vaultMap), &v1alpha1.VaultMap{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultMap), err
}

// Delete takes name of the vaultMap and deletes it. Returns an error if one occurs.
func (c *FakeVaultMaps) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(vaultmapsResource, c.ns, name), &v1alpha1.VaultMap{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVaultMaps) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(vaultmapsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.VaultMapList{})
	return err
}

// Patch applies the patch and returns the patched vaultMap.
func (c *FakeVaultMaps) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultMap, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(vaultmapsResource, c.ns, name, data, subresources...), &v1alpha1.VaultMap{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultMap), err
}
