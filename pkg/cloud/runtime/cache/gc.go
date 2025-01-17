package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
)

type gcRequest struct {
	resourceGroup string
	gvk           schema.GroupVersionKind
	key           types.NamespacedName
}

func (c *cache) startGarbageCollector(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("controller", "gc") // TODO: consider if to use something different than controller
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("Starting garbage collector queue")
	c.garbageCollectorQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	go func() {
		<-ctx.Done()
		c.garbageCollectorQueue.ShutDown()
	}()

	workers := 0
	go func() {
		log.Info("Starting garbage collector workers", "count", c.garbageCollectorConcurrency)
		wg := &sync.WaitGroup{}
		wg.Add(c.garbageCollectorConcurrency)
		for i := 0; i < c.garbageCollectorConcurrency; i++ {
			go func() {
				workers++
				defer wg.Done()
				for c.processGarbageCollectorWorkItem(ctx) { //nolint:revive
				}
			}()
		}
		<-ctx.Done()
		wg.Wait()
	}()

	if err := wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 5*time.Second, false, func(ctx context.Context) (done bool, err error) {
		if workers < c.garbageCollectorConcurrency {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("failed to start garbage collector workers: %v", err)
	}
	return nil
}

func (c *cache) processGarbageCollectorWorkItem(ctx context.Context) bool {
	log := ctrl.LoggerFrom(ctx)

	item, shutdown := c.garbageCollectorQueue.Get()
	if shutdown {
		return false
	}

	// TODO(Fabrizio): Why are we calling the same in defer and directly
	defer func() {
		c.garbageCollectorQueue.Done(item)
	}()
	c.garbageCollectorQueue.Done(item)

	gcr, ok := item.(gcRequest)
	if !ok {
		c.garbageCollectorQueue.Forget(item)
		return true
	}

	deleted, err := c.tryDelete(gcr.resourceGroup, gcr.gvk, gcr.key)
	if err != nil {
		log.Error(err, "Error garbage collecting object", "resourceGroup", gcr.resourceGroup, gcr.gvk.Kind, gcr.key)
	}

	if err == nil && deleted {
		c.garbageCollectorQueue.Forget(item)
		log.Info("Object garbage collected", "resourceGroup", gcr.resourceGroup, gcr.gvk.Kind, gcr.key)
		return true
	}

	c.garbageCollectorQueue.Forget(item)

	requeueAfter := wait.Jitter(c.garbageCollectorRequeueAfter, c.garbageCollectorRequeueAfterJitterFactor)
	c.garbageCollectorQueue.AddAfter(item, requeueAfter)
	return true
}
