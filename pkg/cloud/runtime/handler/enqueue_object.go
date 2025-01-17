package handler

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cevent "github.com/fabriziopandini/cluster-api-provider-goofy/pkg/cloud/runtime/event"
	creconciler "github.com/fabriziopandini/cluster-api-provider-goofy/pkg/cloud/runtime/reconcile"
)

var _ EventHandler = &EnqueueRequestForObject{}

// EnqueueRequestForObject enqueues a Request containing the ResourceGroup, Name and Namespace of the object that is the source of the Event.
// handler.EnqueueRequestForObject is used by almost all Controllers that have associated Resources to reconcile.
type EnqueueRequestForObject struct{}

// Create implements EventHandler.
func (e *EnqueueRequestForObject) Create(evt cevent.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(creconciler.Request{
		ResourceGroup: evt.ResourceGroup,
		NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      evt.Object.GetName(),
		},
	})
}

// Update implements EventHandler.
func (e *EnqueueRequestForObject) Update(evt cevent.UpdateEvent, q workqueue.RateLimitingInterface) {
	switch {
	case evt.ObjectNew != nil:
		q.Add(creconciler.Request{
			ResourceGroup: evt.ResourceGroup,
			NamespacedName: types.NamespacedName{
				Namespace: evt.ObjectNew.GetNamespace(),
				Name:      evt.ObjectNew.GetName(),
			},
		})
	case evt.ObjectOld != nil:
		if evt.ObjectNew != nil && client.ObjectKeyFromObject(evt.ObjectNew) == client.ObjectKeyFromObject(evt.ObjectOld) {
			return
		}
		q.Add(creconciler.Request{
			ResourceGroup: evt.ResourceGroup,
			NamespacedName: types.NamespacedName{
				Namespace: evt.ObjectOld.GetNamespace(),
				Name:      evt.ObjectOld.GetName(),
			},
		})
	}
}

// Delete implements EventHandler.
func (e *EnqueueRequestForObject) Delete(evt cevent.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(creconciler.Request{
		ResourceGroup: evt.ResourceGroup,
		NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      evt.Object.GetName(),
		},
	})
}

// Generic implements EventHandler.
func (e *EnqueueRequestForObject) Generic(evt cevent.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(creconciler.Request{
		ResourceGroup: evt.ResourceGroup,
		NamespacedName: types.NamespacedName{
			Namespace: evt.Object.GetNamespace(),
			Name:      evt.Object.GetName(),
		},
	})
}
