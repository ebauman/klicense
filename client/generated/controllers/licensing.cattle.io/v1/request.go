/*
Copyright 2022.

All Rights Reserved
*/
// Code generated by main. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/ebauman/klicense/api/v1"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type RequestHandler func(string, *v1.Request) (*v1.Request, error)

type RequestController interface {
	generic.ControllerMeta
	RequestClient

	OnChange(ctx context.Context, name string, sync RequestHandler)
	OnRemove(ctx context.Context, name string, sync RequestHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() RequestCache
}

type RequestClient interface {
	Create(*v1.Request) (*v1.Request, error)
	Update(*v1.Request) (*v1.Request, error)
	UpdateStatus(*v1.Request) (*v1.Request, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1.Request, error)
	List(namespace string, opts metav1.ListOptions) (*v1.RequestList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Request, err error)
}

type RequestCache interface {
	Get(namespace, name string) (*v1.Request, error)
	List(namespace string, selector labels.Selector) ([]*v1.Request, error)

	AddIndexer(indexName string, indexer RequestIndexer)
	GetByIndex(indexName, key string) ([]*v1.Request, error)
}

type RequestIndexer func(obj *v1.Request) ([]string, error)

type requestController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewRequestController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) RequestController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &requestController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromRequestHandlerToHandler(sync RequestHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1.Request
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1.Request))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *requestController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1.Request))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateRequestDeepCopyOnChange(client RequestClient, obj *v1.Request, handler func(obj *v1.Request) (*v1.Request, error)) (*v1.Request, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *requestController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *requestController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *requestController) OnChange(ctx context.Context, name string, sync RequestHandler) {
	c.AddGenericHandler(ctx, name, FromRequestHandlerToHandler(sync))
}

func (c *requestController) OnRemove(ctx context.Context, name string, sync RequestHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromRequestHandlerToHandler(sync)))
}

func (c *requestController) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *requestController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *requestController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *requestController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *requestController) Cache() RequestCache {
	return &requestCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *requestController) Create(obj *v1.Request) (*v1.Request, error) {
	result := &v1.Request{}
	return result, c.client.Create(context.TODO(), obj.Namespace, obj, result, metav1.CreateOptions{})
}

func (c *requestController) Update(obj *v1.Request) (*v1.Request, error) {
	result := &v1.Request{}
	return result, c.client.Update(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *requestController) UpdateStatus(obj *v1.Request) (*v1.Request, error) {
	result := &v1.Request{}
	return result, c.client.UpdateStatus(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *requestController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *requestController) Get(namespace, name string, options metav1.GetOptions) (*v1.Request, error) {
	result := &v1.Request{}
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *requestController) List(namespace string, opts metav1.ListOptions) (*v1.RequestList, error) {
	result := &v1.RequestList{}
	return result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *requestController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *requestController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v1.Request, error) {
	result := &v1.Request{}
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type requestCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *requestCache) Get(namespace, name string) (*v1.Request, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1.Request), nil
}

func (c *requestCache) List(namespace string, selector labels.Selector) (ret []*v1.Request, err error) {

	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Request))
	})

	return ret, err
}

func (c *requestCache) AddIndexer(indexName string, indexer RequestIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1.Request))
		},
	}))
}

func (c *requestCache) GetByIndex(indexName, key string) (result []*v1.Request, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1.Request, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1.Request))
	}
	return result, nil
}

type RequestStatusHandler func(obj *v1.Request, status v1.RequestStatus) (v1.RequestStatus, error)

type RequestGeneratingHandler func(obj *v1.Request, status v1.RequestStatus) ([]runtime.Object, v1.RequestStatus, error)

func RegisterRequestStatusHandler(ctx context.Context, controller RequestController, condition condition.Cond, name string, handler RequestStatusHandler) {
	statusHandler := &requestStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromRequestHandlerToHandler(statusHandler.sync))
}

func RegisterRequestGeneratingHandler(ctx context.Context, controller RequestController, apply apply.Apply,
	condition condition.Cond, name string, handler RequestGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &requestGeneratingHandler{
		RequestGeneratingHandler: handler,
		apply:                    apply,
		name:                     name,
		gvk:                      controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterRequestStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type requestStatusHandler struct {
	client    RequestClient
	condition condition.Cond
	handler   RequestStatusHandler
}

func (a *requestStatusHandler) sync(key string, obj *v1.Request) (*v1.Request, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type requestGeneratingHandler struct {
	RequestGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *requestGeneratingHandler) Remove(key string, obj *v1.Request) (*v1.Request, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.Request{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *requestGeneratingHandler) Handle(obj *v1.Request, status v1.RequestStatus) (v1.RequestStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.RequestGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
