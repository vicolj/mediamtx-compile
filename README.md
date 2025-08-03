# MediaMTX 源码编译完整指南

## 项目资源
- **GitHub 仓库**: [mediamtx-compile](https://github.com/vicolj/mediamtx-compile)
- **相关引用**:
  - [不合规 RTSP Content-Base 支持](https://github.com/bluenviron/mediamtx/issues/825)
  - [Go 1.24.0 Android CGO 编译解决方案](https://github.com/wlynxg/anet)

## 目录
1. [环境准备](#环境准备)
2. [支持不合规 RTSP Content-Base](#支持不合规-rtsp-content-base)
3. [编译为共享库](#编译为共享库)
4. [Android 交叉编译](#android-交叉编译)

## 环境准备

### 基础环境配置
```bash
# 下载源码
git clone https://github.com/bluenviron/mediamtx

# 安装 Go 1.24.0
wget https://golang.org/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
```

### 环境变量配置 (~/.bashrc)
```bash
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
```

### 初始化项目
```bash
source ~/.bashrc
go version
cd mediamtx
go clean -modcache
go mod download
```

## 支持不合规 RTSP Content-Base

### 问题描述
当拉取某些摄像头的 RTSP 流时，可能会遇到错误：
```
[rtsp source] ERR: invalid Content-Base: '[X.X.X.6:8554/]'
```

### 解决方案
修改 MediaMTX 依赖的 RTSP 库以放松校验。

### 修改步骤
1. 获取依赖库：
```bash
git clone https://github.com/bluenviron/gortsplib
```

2. 修改 `gortsplib/client.go`：
```go
// 在 findBaseURL 函数中添加特殊处理
func findBaseURL(sd *sdp.SessionDescription, res *base.Response, u *base.URL) (*base.URL, error) {
    // ... 其他代码保持不变 ...

    // use Content-Base
    if cb, ok := res.Header["Content-Base"]; ok {
        if len(cb) != 1 {
            return nil, fmt.Errorf("invalid Content-Base: '%v'", cb)
        }

        if strings.HasPrefix(cb[0], "/") {
            // parse as a relative path
            ret, err := base.ParseURL(u.Scheme + "://" + u.Host + cb[0])
            if err != nil {
                return nil, fmt.Errorf("invalid Content-Base: '%v'", cb)
            }

            // add credentials
            ret.User = u.User

            return ret, nil
        }

        // 新增处理不合规格式的逻辑
        if strings.Contains(cb[0], ":") && strings.HasSuffix(cb[0], "/") && !strings.HasPrefix(cb[0], "rtsp://") {
            cb[0] = "rtsp://" + cb[0]
        }

        ret, err := base.ParseURL(cb[0])
        if err != nil {
            return nil, fmt.Errorf("invalid Content-Base: '%v'", cb)
        }

        // add credentials
        ret.User = u.User

        return ret, nil
    }

    // ... 其他代码保持不变 ...
}
```

3. 修改 go.mod：
```go
replace github.com/bluenviron/gortsplib/v4 => /path/to/local/gortsplib
```

4. 重新编译：
```bash
go generate ./...
CGO_ENABLED=0 go build .
```

## 编译为共享库

### 关键文件配置 (SO)
1. 创建 `export.go`：
```go
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
    
    goConfigPath := C.GoString(configPath)
    s, ok := core.New([]string{goConfigPath})
    if !ok {
        return C.CString("Failed to start MediaMTX")
    }
    
    serverInstance = s
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
    
    serverInstance.Close()
    serverInstance = nil
    return C.CString("MediaMTX stopped successfully")
}
```

2. 重命名主文件  main.go：
```bash
mv main.go main.go.bak
```

### 编译命令
```bash
# 生成动态链接库
go build -o libmediamtx.so -buildmode=c-shared
```

## Android 交叉编译

### NDK 环境配置
```bash
export NDK=/opt/android-ndk-r29-beta2
export TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin
export CC=$TOOLCHAIN/aarch64-linux-android21-clang
export CXX=$TOOLCHAIN/aarch64-linux-android21-clang++
```

### 特殊编译参数
```bash
export CGO_ENABLED=1
export GOOS=android
export GOARCH=arm64

# 编译命令
go build -o libmediamtx_android.so -buildmode=c-shared -ldflags "-checklinkname=0"
```

> **注意**：`-ldflags "-checklinkname=0"` 参数用于解决 Go 1.23.0+ 与 Android 的网络库兼容性问题# mediamtx-compile
