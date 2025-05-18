package resource

import (
	"context"
	"database/sql"
	"net"
	"os"
)

// ContextResourceTracker 是一个与上下文关联的资源追踪器
type ContextResourceTracker struct {
	*ResourceTracker
	ctx context.Context
}

// WithTrackerContext 创建一个与上下文关联的资源追踪器
func WithTrackerContext(ctx context.Context, tracker *ResourceTracker) *ContextResourceTracker {
	return &ContextResourceTracker{
		ResourceTracker: tracker,
		ctx:             ctx,
	}
}

// TrackFile 追踪文件资源
func (ct *ContextResourceTracker) TrackFile(file *os.File) *FileResource {
	resource := ct.ResourceTracker.TrackFile(file)
	if resource != nil {
		go ct.watchContext(resource.ID())
	}
	return resource
}

// TrackNetwork 追踪网络资源
func (ct *ContextResourceTracker) TrackNetwork(conn net.Conn) *NetworkResource {
	resource := ct.ResourceTracker.TrackNetwork(conn)
	if resource != nil {
		go ct.watchContext(resource.ID())
	}
	return resource
}

// TrackDatabase 追踪数据库资源
func (ct *ContextResourceTracker) TrackDatabase(db *sql.DB, connInfo string) *DatabaseResource {
	resource := ct.ResourceTracker.TrackDatabase(db, connInfo)
	if resource != nil {
		go ct.watchContext(resource.ID())
	}
	return resource
}

// TrackGeneric 追踪通用资源
func (ct *ContextResourceTracker) TrackGeneric(id, resourceType string, closer func() error) *GenericResource {
	resource := ct.ResourceTracker.TrackGeneric(id, resourceType, closer)
	if resource != nil {
		go ct.watchContext(resource.ID())
	}
	return resource
}

// watchContext 监视上下文，在上下文取消时释放资源
func (ct *ContextResourceTracker) watchContext(resourceID string) {
	<-ct.ctx.Done()
	ct.ResourceTracker.Release(resourceID)
}
