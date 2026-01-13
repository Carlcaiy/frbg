# WebSocket 调用健康检查计划

## 信息收集

### 1. WebSocket架构概述
- **WebSocket库**: 使用 `github.com/gobwas/ws`
- **WebSocket管理器**: `core/conn.go` 中的 `WsConn` 结构体
- **消息编解码**: `codec/parser.go` 中的 `WsRead`/`WsWrite` 函数
- **连接处理**: `core/epoll.go` 中的 `processAcceptWebSocket()` 函数

### 2. 日志中发现的问题
- 出现 "websocket read error: EOF" 错误
- 存在 "i/o timeout" 超时问题
- 客户端连接频繁断开重连
- 部分用户连接丢失（not find user）

### 3. 需要检查的关键组件
- [ ] WebSocket握手升级过程
- [ ] 心跳机制实现
- [ ] 消息读写超时设置
- [ ] 连接管理（心跳、超时检测）
- [ ] 错误处理逻辑

## 详细检查计划

### 阶段1: WebSocket握手和连接建立
1. 检查 `processAcceptWebSocket()` 函数
   - [ ] 连接数限制是否合理
   - [ ] WebSocket升级是否成功
   - [ ] FD重复检测逻辑

### 阶段2: 消息读写机制
1. 检查 `WsConn.Read()` 和 `WsConn.Write()`
   - [ ] 超时设置是否合理（当前100ms读取超时）
   - [ ] 错误处理是否完善
   - [ ] 是否有内存泄漏风险

### 阶段3: 心跳和超时检测
1. 检查心跳机制
   - [ ] 心跳间隔是否合理
   - [ ] 超时检测是否及时
   - [ ] 心跳包格式是否正确

### 阶段4: 连接生命周期管理
1. 检查连接关闭流程
   - [ ] `Del()` 函数是否正确清理资源
   - [ ] `Close()` 回调是否执行
   - [ ] 客户端断开时的处理

## 依赖文件
- `core/epoll.go`: 主事件循环
- `core/conn.go`: 连接实现
- `codec/parser.go`: 消息编解码
- `examples/gate/route/route.go`: 业务路由
- `examples/wsclient/main.go`: 测试客户端

## 后续步骤
1. 执行代码检查
2. 运行测试验证
3. 性能分析
4. 生成优化建议
