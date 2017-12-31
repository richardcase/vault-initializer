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
package v1alpha1

import (
	v1alpha1 "github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	scheme "github.com/richardcase/vault-initializer/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// VaultMapsGetter has a method to return a VaultMapInterface.
// A group's client should implement this interface.
type VaultMapsGetter interface {
	VaultMaps(namespace string) VaultMapInterface
}

// VaultMapInterface has methods to work with VaultMap resources.
type VaultMapInterface interface {
	Create(*v1alpha1.VaultMap) (*v1alpha1.VaultMap, error)
	Update(*v1alpha1.VaultMap) (*v1alpha1.VaultMap, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.VaultMap, error)
	List(opts v1.ListOptions) (*v1alpha1.VaultMapList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultMap, err error)
	VaultMapExpansion
}

// vaultMaps implements VaultMapInterface
type vaultMaps struct {
	client rest.Interface
	ns     string
}

// newVaultMaps returns a VaultMaps
func newVaultMaps(c *VaultinitV1alpha1Client, namespace string) *vaultMaps {
	return &vaultMaps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the vaultMap, and returns the corresponding vaultMap object, and an error if there is any.
func (c *vaultMaps) Get(name string, options v1.GetOptions) (result *v1alpha1.VaultMap, err error) {
	result = &v1alpha1.VaultMap{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vaultmaps").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VaultMaps that match those selectors.
func (c *vaultMaps) List(opts v1.ListOptions) (result *v1alpha1.VaultMapList, err error) {
	result = &v1alpha1.VaultMapList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vaultmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested vaultMaps.
func (c *vaultMaps) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("vaultmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a vaultMap and creates it.  Returns the server's representation of the vaultMap, and an error, if there is any.
func (c *vaultMaps) Create(vaultMap *v1alpha1.VaultMap) (result *v1alpha1.VaultMap, err error) {
	result = &v1alpha1.VaultMap{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("vaultmaps").
		Body(vaultMap).
		Do().
		Into(result)
	return
}

// Update takes the representation of a vaultMap and updates it. Returns the server's representation of the vaultMap, and an error, if there is any.
func (c *vaultMaps) Update(vaultMap *v1alpha1.VaultMap) (result *v1alpha1.VaultMap, err error) {
	result = &v1alpha1.VaultMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("vaultmaps").
		Name(vaultMap.Name).
		Body(vaultMap).
		Do().
		Into(result)
	return
}

// Delete takes name of the vaultMap and deletes it. Returns an error if one occurs.
func (c *vaultMaps) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vaultmaps").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *vaultMaps) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vaultmaps").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched vaultMap.
func (c *vaultMaps) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultMap, err error) {
	result = &v1alpha1.VaultMap{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("vaultmaps").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
