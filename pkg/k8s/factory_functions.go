// Copyright 2018-2020 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"reflect"

	"github.com/cilium/cilium/pkg/annotation"
	"github.com/cilium/cilium/pkg/comparator"
	"github.com/cilium/cilium/pkg/datapath"
	cilium_v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slim_corev1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/core/v1"
	"github.com/cilium/cilium/pkg/k8s/types"
	"github.com/cilium/cilium/pkg/logging/logfields"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
)

func ObjToV1NetworkPolicy(obj interface{}) *types.NetworkPolicy {
	k8sNP, ok := obj.(*types.NetworkPolicy)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 NetworkPolicy")
		return nil
	}
	return k8sNP
}

func ObjToV1Services(obj interface{}) *types.Service {
	svc, ok := obj.(*types.Service)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 Service")
		return nil
	}
	return svc
}

func ObjToV1Endpoints(obj interface{}) *types.Endpoints {
	ep, ok := obj.(*types.Endpoints)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 Endpoints")
		return nil
	}
	return ep
}

func ObjToV1EndpointSlice(obj interface{}) *types.EndpointSlice {
	ep, ok := obj.(*types.EndpointSlice)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 EndpointSlice")
		return nil
	}
	return ep
}

func ObjToSlimCNP(obj interface{}) *types.SlimCNP {
	cnp, ok := obj.(*types.SlimCNP)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v2 CiliumNetworkPolicy")
		return nil
	}
	return cnp
}

func ObjTov1Pod(obj interface{}) *slim_corev1.Pod {
	pod, ok := obj.(*slim_corev1.Pod)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 Pod")
		return nil
	}
	return pod
}

func ObjToV1Node(obj interface{}) *types.Node {
	node, ok := obj.(*types.Node)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 Node")
		return nil
	}
	return node
}

func ObjToV1Namespace(obj interface{}) *types.Namespace {
	ns, ok := obj.(*types.Namespace)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid k8s v1 Namespace")
		return nil
	}
	return ns
}

func EqualV1NetworkPolicy(np1, np2 *types.NetworkPolicy) bool {
	// As Cilium uses all of the Spec from a NP it's not probably not worth
	// it to create a dedicated deep equal	 function to compare both network
	// policies.
	return np1.Name == np2.Name &&
		np1.Namespace == np2.Namespace &&
		reflect.DeepEqual(np1.Spec, np2.Spec)
}

func EqualV1Services(k8sSVC1, k8sSVC2 *types.Service, nodeAddressing datapath.NodeAddressing) bool {
	// Service annotations are used to mark services as global, shared, etc.
	if !comparator.MapStringEquals(k8sSVC1.GetAnnotations(), k8sSVC2.GetAnnotations()) {
		return false
	}

	svcID1, svc1 := ParseService(k8sSVC1, nodeAddressing)
	svcID2, svc2 := ParseService(k8sSVC2, nodeAddressing)

	if svcID1 != svcID2 {
		return false
	}

	// Please write all the equalness logic inside the K8sServiceInfo.Equals()
	// method.
	return svc1.DeepEquals(svc2)
}

func EqualV1Endpoints(ep1, ep2 *types.Endpoints) bool {
	// We only care about the Name, Namespace and Subsets of a particular
	// endpoint.
	return ep1.Name == ep2.Name &&
		ep1.Namespace == ep2.Namespace &&
		reflect.DeepEqual(ep1.Subsets, ep2.Subsets)
}

func EqualV1EndpointSlice(ep1, ep2 *types.EndpointSlice) bool {
	// We only care about the Name, Namespace and Subsets of a particular
	// endpoint.
	// AddressType is omitted because it's immutable after EndpointSlice's
	// creation.
	return ep1.Name == ep2.Name &&
		ep1.Namespace == ep2.Namespace &&
		reflect.DeepEqual(ep1.Endpoints, ep2.Endpoints) &&
		reflect.DeepEqual(ep1.Ports, ep2.Ports)
}

func EqualV2CNP(cnp1, cnp2 *types.SlimCNP) bool {
	if !(cnp1.Name == cnp2.Name && cnp1.Namespace == cnp2.Namespace) {
		return false
	}

	// Ignore v1.LastAppliedConfigAnnotation annotation
	lastAppliedCfgAnnotation1, ok1 := cnp1.GetAnnotations()[v1.LastAppliedConfigAnnotation]
	lastAppliedCfgAnnotation2, ok2 := cnp2.GetAnnotations()[v1.LastAppliedConfigAnnotation]
	defer func() {
		if ok1 && cnp1.GetAnnotations() != nil {
			cnp1.GetAnnotations()[v1.LastAppliedConfigAnnotation] = lastAppliedCfgAnnotation1
		}
		if ok2 && cnp2.GetAnnotations() != nil {
			cnp2.GetAnnotations()[v1.LastAppliedConfigAnnotation] = lastAppliedCfgAnnotation2
		}
	}()
	delete(cnp1.GetAnnotations(), v1.LastAppliedConfigAnnotation)
	delete(cnp2.GetAnnotations(), v1.LastAppliedConfigAnnotation)

	return comparator.MapStringEquals(cnp1.GetAnnotations(), cnp2.GetAnnotations()) &&
		reflect.DeepEqual(cnp1.Spec, cnp2.Spec) &&
		reflect.DeepEqual(cnp1.Specs, cnp2.Specs)
}

// AnnotationsEqual returns whether the annotation with any key in
// relevantAnnotations is equal in anno1 and anno2.
func AnnotationsEqual(relevantAnnotations []string, anno1, anno2 map[string]string) bool {
	for _, an := range relevantAnnotations {
		if anno1[an] != anno2[an] {
			return false
		}
	}
	return true
}

func EqualV1PodContainers(c1, c2 slim_corev1.Container) bool {
	if c1.Name != c2.Name ||
		c1.Image != c2.Image {
		return false
	}

	if len(c1.VolumeMounts) != len(c2.VolumeMounts) {
		return false
	}
	for i, vlmMount1 := range c1.VolumeMounts {
		if vlmMount1.MountPath != c2.VolumeMounts[i].MountPath {
			return false
		}
	}
	return true
}

// EqualPodIPs returns true if both slices are equal.
func EqualPodIPs(a, b []slim_corev1.PodIP) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func EqualV1Pod(pod1, pod2 *slim_corev1.Pod) bool {
	// We only care about the HostIP, the PodIP and the labels of the pods.
	if pod1.Status.PodIP != pod2.Status.PodIP ||
		pod1.Status.HostIP != pod2.Status.HostIP ||
		pod1.Spec.ServiceAccountName != pod2.Spec.ServiceAccountName ||
		pod1.Spec.HostNetwork != pod2.Spec.HostNetwork {
		return false
	}

	if !EqualPodIPs(pod1.Status.PodIPs, pod2.Status.PodIPs) {
		return false
	}

	if !AnnotationsEqual([]string{annotation.ProxyVisibility}, pod1.Annotations, pod2.Annotations) {
		return false
	}

	oldPodLabels := pod1.Labels
	newPodLabels := pod2.Labels
	if !comparator.MapStringEquals(oldPodLabels, newPodLabels) {
		return false
	}

	if len(pod1.Spec.Containers) != len(pod2.Spec.Containers) {
		return false
	}
	for i, c1 := range pod1.Spec.Containers {
		if !EqualV1PodContainers(c1, pod2.Spec.Containers[i]) {
			return false
		}
	}
	return true
}

func EqualV1Node(node1, node2 *types.Node) bool {
	if node1.GetObjectMeta().GetName() != node2.GetObjectMeta().GetName() {
		return false
	}

	anno1 := node1.GetAnnotations()
	anno2 := node2.GetAnnotations()
	annotationsWeCareAbout := []string{
		annotation.CiliumHostIP,
		annotation.CiliumHostIPv6,
		annotation.V4HealthName,
		annotation.V6HealthName,
	}
	for _, an := range annotationsWeCareAbout {
		if anno1[an] != anno2[an] {
			return false
		}
	}
	if len(node1.SpecTaints) != len(node2.SpecTaints) {
		return false
	}
	for i, taint2 := range node2.SpecTaints {
		taint1 := node1.SpecTaints[i]
		if !taint1.MatchTaint(&taint2) {
			return false
		}
		if taint1.Value != taint2.Value {
			return false
		}
		if !taint1.TimeAdded.Equal(taint2.TimeAdded) {
			return false
		}
	}
	return true
}

func EqualV1Namespace(ns1, ns2 *types.Namespace) bool {
	// we only care about namespace labels.
	return ns1.Name == ns2.Name &&
		comparator.MapStringEquals(ns1.GetLabels(), ns2.GetLabels())
}

// ConvertToNetworkPolicy converts a *networkingv1.NetworkPolicy into a
// *types.NetworkPolicy or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.NetworkPolicy in its Obj.
// If the given obj can't be cast into either *networkingv1.NetworkPolicy
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToNetworkPolicy(obj interface{}) interface{} {
	// TODO check which fields we really need
	switch concreteObj := obj.(type) {
	case *networkingv1.NetworkPolicy:
		return &types.NetworkPolicy{
			NetworkPolicy: concreteObj,
		}
	case cache.DeletedFinalStateUnknown:
		netPol, ok := concreteObj.Obj.(*networkingv1.NetworkPolicy)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.NetworkPolicy{
				NetworkPolicy: netPol,
			},
		}
	default:
		return obj
	}
}

// ConvertToK8sService converts a *v1.Service into a
// *types.Service or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.Service in its Obj.
// If the given obj can't be cast into either *v1.Service
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToK8sService(obj interface{}) interface{} {
	// TODO check which fields we really need
	switch concreteObj := obj.(type) {
	case *v1.Service:
		return &types.Service{
			Service: concreteObj,
		}
	case cache.DeletedFinalStateUnknown:
		svc, ok := concreteObj.Obj.(*v1.Service)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.Service{
				Service: svc,
			},
		}
	default:
		return obj
	}
}

// ConvertToK8sEndpoints converts a *v1.Endpoints into a
// *types.Endpoints or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.Endpoints in its Obj.
// If the given obj can't be cast into either *v1.Endpoints
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToK8sEndpoints(obj interface{}) interface{} {
	// TODO check which fields we really need
	switch concreteObj := obj.(type) {
	case *v1.Endpoints:
		return &types.Endpoints{
			Endpoints: concreteObj,
		}
	case cache.DeletedFinalStateUnknown:
		eps, ok := concreteObj.Obj.(*v1.Endpoints)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.Endpoints{
				Endpoints: eps,
			},
		}
	default:
		return obj
	}
}

// ConvertToK8sEndpointSlice converts a *v1beta1.EndpointSlice into a
// *types.Endpoints or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.Endpoints in its Obj.
// If the given obj can't be cast into either *v1.Endpoints
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToK8sEndpointSlice(obj interface{}) interface{} {
	// TODO check which fields we really need
	switch concreteObj := obj.(type) {
	case *v1beta1.EndpointSlice:
		return &types.EndpointSlice{
			EndpointSlice: concreteObj,
		}
	case cache.DeletedFinalStateUnknown:
		eps, ok := concreteObj.Obj.(*v1beta1.EndpointSlice)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.EndpointSlice{
				EndpointSlice: eps,
			},
		}
	default:
		return obj
	}
}

// ConvertToCCNPWithStatus converts a *cilium_v2.CiliumClusterwideNetworkPolicy
// into *types.SlimCNP or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.SlimCNP in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumClusterwideNetworkPolicy
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToCCNPWithStatus(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumClusterwideNetworkPolicy:
		t := &types.SlimCNP{
			CiliumNetworkPolicy: concreteObj.CiliumNetworkPolicy,
		}
		t.Status = concreteObj.Status
		return t

	case cache.DeletedFinalStateUnknown:
		cnp, ok := concreteObj.Obj.(*cilium_v2.CiliumClusterwideNetworkPolicy)
		if !ok {
			return obj
		}
		t := &types.SlimCNP{
			CiliumNetworkPolicy: cnp.CiliumNetworkPolicy,
		}
		t.Status = cnp.Status
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: t,
		}

	default:
		return obj
	}
}

// ConvertToCNPWithStatus converts a *cilium_v2.CiliumNetworkPolicy or a
// *cilium_v2.CiliumClusterwideNetworkPolicy into a
// *types.SlimCNP or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.SlimCNP in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumNetworkPolicy
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToCNPWithStatus(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumNetworkPolicy:
		return &types.SlimCNP{
			CiliumNetworkPolicy: concreteObj,
		}
	case cache.DeletedFinalStateUnknown:
		cnp, ok := concreteObj.Obj.(*cilium_v2.CiliumNetworkPolicy)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.SlimCNP{
				CiliumNetworkPolicy: cnp,
			},
		}
	default:
		return obj
	}
}

// ConvertToCCNP converts a *cilium_v2.CiliumClusterwideNetworkPolicy into a
// *types.SlimCNP without the Status field of the given CNP, or a
// cache.DeletedFinalStateUnknown into a cache.DeletedFinalStateUnknown with a
// *types.SlimCNP, also without the Status field of the given CNP, in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumClusterwideNetworkPolicy
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
// WARNING calling this function will set *all* fields of the given CNP as
// empty.
func ConvertToCCNP(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumClusterwideNetworkPolicy:
		cnp := &types.SlimCNP{
			CiliumNetworkPolicy: &cilium_v2.CiliumNetworkPolicy{
				TypeMeta:   concreteObj.TypeMeta,
				ObjectMeta: concreteObj.ObjectMeta,
				Spec:       concreteObj.Spec,
				Specs:      concreteObj.Specs,
			},
		}
		*concreteObj = cilium_v2.CiliumClusterwideNetworkPolicy{}
		return cnp

	case cache.DeletedFinalStateUnknown:
		cnp, ok := concreteObj.Obj.(*cilium_v2.CiliumClusterwideNetworkPolicy)
		if !ok {
			return obj
		}
		dfsu := cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.SlimCNP{
				CiliumNetworkPolicy: &cilium_v2.CiliumNetworkPolicy{
					TypeMeta:   cnp.TypeMeta,
					ObjectMeta: cnp.ObjectMeta,
					Spec:       cnp.Spec,
					Specs:      cnp.Specs,
				},
			},
		}
		*cnp = cilium_v2.CiliumClusterwideNetworkPolicy{}
		return dfsu

	default:
		return obj
	}
}

// ConvertToCNP converts a *cilium_v2.CiliumNetworkPolicy into a
// *types.SlimCNP without the Status field of the given CNP, or a
// cache.DeletedFinalStateUnknown into a cache.DeletedFinalStateUnknown with a
// *types.SlimCNP, also without the Status field of the given CNP, in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumNetworkPolicy
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
// WARNING calling this function will set *all* fields of the given CNP as
// empty.
func ConvertToCNP(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumNetworkPolicy:
		cnp := &types.SlimCNP{
			CiliumNetworkPolicy: &cilium_v2.CiliumNetworkPolicy{
				TypeMeta:   concreteObj.TypeMeta,
				ObjectMeta: concreteObj.ObjectMeta,
				Spec:       concreteObj.Spec,
				Specs:      concreteObj.Specs,
			},
		}
		*concreteObj = cilium_v2.CiliumNetworkPolicy{}
		return cnp
	case cache.DeletedFinalStateUnknown:
		cnp, ok := concreteObj.Obj.(*cilium_v2.CiliumNetworkPolicy)
		if !ok {
			return obj
		}
		dfsu := cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.SlimCNP{
				CiliumNetworkPolicy: &cilium_v2.CiliumNetworkPolicy{
					TypeMeta:   cnp.TypeMeta,
					ObjectMeta: cnp.ObjectMeta,
					Spec:       cnp.Spec,
					Specs:      cnp.Specs,
				},
			},
		}
		*cnp = cilium_v2.CiliumNetworkPolicy{}
		return dfsu
	default:
		return obj
	}
}

// ConvertToNode converts a *v1.Node into a
// *types.Node or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.Node in its Obj.
// If the given obj can't be cast into either *v1.Node
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
// WARNING calling this function will set *all* fields of the given Node as
// empty.
func ConvertToNode(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *v1.Node:
		p := &types.Node{
			TypeMeta:        concreteObj.TypeMeta,
			ObjectMeta:      concreteObj.ObjectMeta,
			StatusAddresses: concreteObj.Status.Addresses,
			SpecPodCIDR:     concreteObj.Spec.PodCIDR,
			SpecPodCIDRs:    concreteObj.Spec.PodCIDRs,
			SpecTaints:      concreteObj.Spec.Taints,
		}
		*concreteObj = v1.Node{}
		return p
	case cache.DeletedFinalStateUnknown:
		node, ok := concreteObj.Obj.(*v1.Node)
		if !ok {
			return obj
		}
		dfsu := cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.Node{
				TypeMeta:        node.TypeMeta,
				ObjectMeta:      node.ObjectMeta,
				StatusAddresses: node.Status.Addresses,
				SpecPodCIDR:     node.Spec.PodCIDR,
				SpecPodCIDRs:    node.Spec.PodCIDRs,
				SpecTaints:      node.Spec.Taints,
			},
		}
		*node = v1.Node{}
		return dfsu
	default:
		return obj
	}
}

// ConvertToNamespace converts a *v1.Namespace into a
// *types.Namespace or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *types.Namespace in its Obj.
// If the given obj can't be cast into either *v1.Namespace
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
// WARNING calling this function will set *all* fields of the given Namespace as
// empty.
func ConvertToNamespace(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *v1.Namespace:
		p := &types.Namespace{
			TypeMeta:   concreteObj.TypeMeta,
			ObjectMeta: concreteObj.ObjectMeta,
		}
		*concreteObj = v1.Namespace{}
		return p
	case cache.DeletedFinalStateUnknown:
		namespace, ok := concreteObj.Obj.(*v1.Namespace)
		if !ok {
			return obj
		}
		dfsu := cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.Namespace{
				TypeMeta:   namespace.TypeMeta,
				ObjectMeta: namespace.ObjectMeta,
			},
		}
		*namespace = v1.Namespace{}
		return dfsu
	default:
		return obj
	}
}

// ConvertToCiliumNode converts a *cilium_v2.CiliumNode into a
// *cilium_v2.CiliumNode or a cache.DeletedFinalStateUnknown into
// a cache.DeletedFinalStateUnknown with a *cilium_v2.CiliumNode in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumNode
// nor cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToCiliumNode(obj interface{}) interface{} {
	// TODO create a slim type of the CiliumNode
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumNode:
		return concreteObj
	case cache.DeletedFinalStateUnknown:
		ciliumNode, ok := concreteObj.Obj.(*cilium_v2.CiliumNode)
		if !ok {
			return obj
		}
		return cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: ciliumNode,
		}
	default:
		return obj
	}
}

// ObjToCiliumNode attempts to cast object to a CiliumNode object and
// returns a deep copy if the castin succeeds. Otherwise, nil is returned.
func ObjToCiliumNode(obj interface{}) *cilium_v2.CiliumNode {
	cn, ok := obj.(*cilium_v2.CiliumNode)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid CiliumNode")
		return nil
	}
	return cn
}

// ConvertToCiliumEndpoint converts a *cilium_v2.CiliumEndpoint into a
// *types.CiliumEndpoint or a cache.DeletedFinalStateUnknown into a
// cache.DeletedFinalStateUnknown with a *types.CiliumEndpoint in its Obj.
// If the given obj can't be cast into either *cilium_v2.CiliumEndpoint nor
// cache.DeletedFinalStateUnknown, the original obj is returned.
func ConvertToCiliumEndpoint(obj interface{}) interface{} {
	switch concreteObj := obj.(type) {
	case *cilium_v2.CiliumEndpoint:
		p := &types.CiliumEndpoint{
			TypeMeta:   concreteObj.TypeMeta,
			ObjectMeta: concreteObj.ObjectMeta,
			Encryption: &concreteObj.Status.Encryption,
			Identity:   concreteObj.Status.Identity,
			Networking: concreteObj.Status.Networking,
			NamedPorts: concreteObj.Status.NamedPorts,
		}
		*concreteObj = cilium_v2.CiliumEndpoint{}
		return p
	case cache.DeletedFinalStateUnknown:
		ciliumEndpoint, ok := concreteObj.Obj.(*cilium_v2.CiliumEndpoint)
		if !ok {
			return obj
		}
		dfsu := cache.DeletedFinalStateUnknown{
			Key: concreteObj.Key,
			Obj: &types.CiliumEndpoint{
				TypeMeta:   ciliumEndpoint.TypeMeta,
				ObjectMeta: ciliumEndpoint.ObjectMeta,
				Encryption: &ciliumEndpoint.Status.Encryption,
				Identity:   ciliumEndpoint.Status.Identity,
				Networking: ciliumEndpoint.Status.Networking,
				NamedPorts: ciliumEndpoint.Status.NamedPorts,
			},
		}
		*ciliumEndpoint = cilium_v2.CiliumEndpoint{}
		return dfsu
	default:
		return obj
	}
}

// ObjToCiliumEndpoint attempts to cast object to a CiliumEndpoint object
// and returns a deep copy if the castin succeeds. Otherwise, nil is returned.
func ObjToCiliumEndpoint(obj interface{}) *types.CiliumEndpoint {
	ce, ok := obj.(*types.CiliumEndpoint)
	if !ok {
		log.WithField(logfields.Object, logfields.Repr(obj)).
			Warn("Ignoring invalid CiliumEndpoint")
		return nil
	}
	return ce
}
