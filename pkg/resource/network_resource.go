package resource

import (
	"fmt"
	"net"
	"time"
)

// NetworkResource 表示一个网络资源
type NetworkResource struct {
	*BaseResource
	conn net.Conn
}

// NewNetworkResource 创建一个新的网络资源
func NewNetworkResource(id string, conn net.Conn) *NetworkResource {
	return &NetworkResource{
		BaseResource: NewBaseResource(id, "network", nil),
		conn:         conn,
	}
}

// Close 关闭网络资源
func (r *NetworkResource) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Conn 返回网络连接对象
func (r *NetworkResource) Conn() net.Conn {
	r.UpdateLastUsed()
	return r.conn
}

// RemoteAddr 返回远程地址
func (r *NetworkResource) RemoteAddr() net.Addr {
	if r.conn != nil {
		return r.conn.RemoteAddr()
	}
	return nil
}

// LocalAddr 返回本地地址
func (r *NetworkResource) LocalAddr() net.Addr {
	if r.conn != nil {
		return r.conn.LocalAddr()
	}
	return nil
}

// TrackNetwork 追踪网络资源
func (rt *ResourceTracker) TrackNetwork(conn net.Conn) *NetworkResource {
	if conn == nil {
		return nil
	}

	id := fmt.Sprintf("network:%d:%s-%s", time.Now().UnixNano(), conn.LocalAddr(), conn.RemoteAddr())
	resource := NewNetworkResource(id, conn)
	rt.Track(resource)
	return resource
}
