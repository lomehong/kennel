package resource

// GenericResource 表示一个通用资源
type GenericResource struct {
	*BaseResource
	data interface{}
}

// NewGenericResource 创建一个新的通用资源
func NewGenericResource(id, resourceType string, closer func() error) *GenericResource {
	return &GenericResource{
		BaseResource: NewBaseResource(id, resourceType, closer),
	}
}

// SetData 设置资源数据
func (r *GenericResource) SetData(data interface{}) {
	r.data = data
	r.UpdateLastUsed()
}

// Data 获取资源数据
func (r *GenericResource) Data() interface{} {
	r.UpdateLastUsed()
	return r.data
}

// TrackGeneric 追踪通用资源
func (rt *ResourceTracker) TrackGeneric(id, resourceType string, closer func() error) *GenericResource {
	resource := NewGenericResource(id, resourceType, closer)
	rt.Track(resource)
	return resource
}
