package main
import (
	"C"
	"sync"
	"github.com/bluenviron/mediamtx/internal/core"
)
var (
	serverInstance *core.Core
	serverMutex    sync.Mutex
)
//export StartMediaMTX
func StartMediaMTX(configPath *C.char) *C.char {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	if serverInstance != nil {
		return C.CString("Server is already running")
	}
	// 将 C 字符串转换为 Go 字符串
	goConfigPath := C.GoString(configPath)
	// 初始化服务器
	s, ok := core.New([]string{goConfigPath})
	if !ok {
		return C.CString("Failed to start MediaMTX")
	}
	serverInstance = s
	// 启动服务器
	go serverInstance.Wait()
	return C.CString("MediaMTX started successfully")
}
//export StopMediaMTX
func StopMediaMTX() *C.char {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	if serverInstance == nil {
		return C.CString("Server is not running")
	}
	// 停止服务器
	serverInstance.Close()
	serverInstance = nil
	return C.CString("MediaMTX stopped successfully")
}
