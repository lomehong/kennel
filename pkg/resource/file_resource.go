package resource

import (
	"fmt"
	"os"
	"time"
)

// FileResource 表示一个文件资源
type FileResource struct {
	*BaseResource
	file *os.File
}

// NewFileResource 创建一个新的文件资源
func NewFileResource(id string, file *os.File) *FileResource {
	return &FileResource{
		BaseResource: NewBaseResource(id, "file", nil),
		file:         file,
	}
}

// Close 关闭文件资源
func (r *FileResource) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// File 返回文件对象
func (r *FileResource) File() *os.File {
	r.UpdateLastUsed()
	return r.file
}

// TrackFile 追踪文件资源
func (rt *ResourceTracker) TrackFile(file *os.File) *FileResource {
	if file == nil {
		return nil
	}

	id := fmt.Sprintf("file:%d:%s", time.Now().UnixNano(), file.Name())
	resource := NewFileResource(id, file)
	rt.Track(resource)
	return resource
}
