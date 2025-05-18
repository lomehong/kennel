package resource

import (
	"database/sql"
	"fmt"
	"time"
)

// DatabaseResource 表示一个数据库资源
type DatabaseResource struct {
	*BaseResource
	db       *sql.DB
	connInfo string
}

// NewDatabaseResource 创建一个新的数据库资源
func NewDatabaseResource(id string, db *sql.DB, connInfo string) *DatabaseResource {
	return &DatabaseResource{
		BaseResource: NewBaseResource(id, "database", nil),
		db:           db,
		connInfo:     connInfo,
	}
}

// Close 关闭数据库资源
func (r *DatabaseResource) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// DB 返回数据库对象
func (r *DatabaseResource) DB() *sql.DB {
	r.UpdateLastUsed()
	return r.db
}

// ConnInfo 返回连接信息
func (r *DatabaseResource) ConnInfo() string {
	return r.connInfo
}

// TrackDatabase 追踪数据库资源
func (rt *ResourceTracker) TrackDatabase(db *sql.DB, connInfo string) *DatabaseResource {
	if db == nil {
		return nil
	}

	id := fmt.Sprintf("database:%d:%s", time.Now().UnixNano(), connInfo)
	resource := NewDatabaseResource(id, db, connInfo)
	rt.Track(resource)
	return resource
}
