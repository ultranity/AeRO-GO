## AeRO-proxy: proxy for RPC behind NAT

可控的TCP端口映射服务

## 目标
- 支持不同网络环境下的分布式连接
- 模块化设计,服务器与客户端功能解耦
- 异步处理,提高并发性能
- 传输加密,确保连接安全

## 使用方式
### TCP流量转发,实现任意TCP协议的穿透
1. 启动 Server节点, 监听特定端口
```
aeros --ip=0.0.0.0 --port <port>

```
2. 启动ProxyWorker,连接ProxyServer

```
aeroc --tag <client tag> --server=<server_ip:port> --target <alias_name@local_ip:local_port>
```
#### 流程
- Worker节点与Server节点建立长连接,Server分配监听端口,注册为工作节点
- Server节点监听端口接收外部连接请求,转发给内网Worker节点
- Worker节点接收请求数据,转发给内网服务
- 内网服务返回响应数据,通过Worker节点转发回Server节点

## 参数
> Usage of AeRO proxy client:
  -tag string
        client name (default "aero")
  -host string
        client host metadata (default: hostname)
  -server string
        server ip:port (default "0.0.0.0:8080")
  -target string
        target name:ip:port list
  -auth string
        server auth code
  -debug
        debug mode
  -log string
        log file
  -ping int
        heartbeat ping interval (default 60)
  -pool int
        pipe pool size estimation (default 1)

> Usage of AeRO proxy server:
    -ip string
            server ip (default "0.0.0.0")
    -port string
            server port (default "8080")
    -auth string
            server auth code
    -debug
            debug mode
    -log string
            log file
    -api string
            server control api (default "localhost:3000")
    -mux string
            http mux server (default "localhost:4000")
    -domain string
            server domain (if set, mux server use sub domain like: <tag>.<name>.<domain> to access target, otherwise use <server_ip>:<port>/<tag>/<name>)

## 子域名映射
当设置 --mux 参数时,Server节点会启动一个http服务,用于子域名映射, 如同时设置 --domain 参数,则会使用子域名的方式映射，如：`<tag>.<name>.<domain>/*` 将被映射为 由 tag 标识的 Worker 节点的 name 别名的服务

否则使用路径参数，如：`localhost:4000/<tag>/<name>/*`

当存在多个相同tag Worker节点时，将会采用roundrubin方案选择一个节点进行负载均衡
### 指定client
server分配的Client ID不重复，同理可以使用 Client ID 代替`<tag>`指定单个Worker节点

为做区分，需在末尾附加#号，即 `<tag> = <cid>@`
## 身份认证
默认为空，使用 --auth 参数设置简易验证，Server节点与Worker节点需要使用相同的认证码才能建立连接

## 管理API
当设置 --api 参数时,
- GET /health
- GET /list 获取所有Worker节点信息
- GET /list?tags=<tag1>,<tag2> 获取对应tag节点信息
- GET /ping?cid=<cid> 主动ping对应cid节点（返回值为发送时间）

## TODO
- [ ] zstd传输压缩
- [ ] 流量统计和管控
- [ ] 日志与监控UI
- [ ] 分配规则控制
- [ ] UDP protocol support
- [ ] libp2p support

## Why another ...
- 应用场景差异
- 自定义API需要
- 学习实践

### 主要借鉴：各类端口转发工具
- frp
- nps
- ngrok
- webtunnel
- sshproxy
- gsocket


### 虚拟子网/VPN
- wireguard
- tailscale/headscale
- zerotier
- netbird

需要复杂的ACL配置,不适合动态分配

### microservice/rpc
- rpcx
- dubbo
- CapnP RPC
- thrift

问题层级不同

## Deps
- go-yamux v4 as connection multiplexer
- fiber as api/http server
- zerolog as logger
- ants as goroutine pool
- netstat from git.mills.io/prologic/go-netstat with pr https://github.com/cakturk/go-netstat/pull/19 merged