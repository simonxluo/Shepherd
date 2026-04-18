// Package api provides Node API adapter for backward compatibility
// 这个包提供了 API 适配层，确保向后兼容性
package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster/scanner"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster/scheduler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/node"
)

// NodeAdapter 将 API 调用适配到 Node 和 Scheduler
type NodeAdapter struct {
	node          *node.Node
	log           *logger.Logger
	scanner       *scanner.Scanner
	scheduler     *scheduler.Scheduler
	eventCallback func(eventType string, data interface{}) // 事件回调函数，用于 WebSocket/SSE 广播
}

// NewNodeAdapter 创建一个新的 Node API 适配器
func NewNodeAdapter(n *node.Node, log *logger.Logger, schedulerCfg *config.SchedulerConfig) *NodeAdapter {
	// 创建 scheduler.ClientManager 适配器，将 Node 适配到 Scheduler 的接口
	clientMgr := &nodeClientManager{node: n}

	// 创建 Scheduler
	sched := scheduler.NewScheduler(schedulerCfg, clientMgr)

	return &NodeAdapter{
		node:      n,
		log:       log,
		scanner:   scanner.NewScanner(&config.NetworkScanConfig{}, log),
		scheduler: sched,
	}
}

// SetEventCallback 设置事件回调函数
func (a *NodeAdapter) SetEventCallback(callback func(eventType string, data interface{})) {
	a.eventCallback = callback
}

// GetScheduler 返回调度器实例（供 server 使用模型加载分发）
func (a *NodeAdapter) GetScheduler() *scheduler.Scheduler {
	return a.scheduler
}

// GetNodeID 返回当前节点 ID
func (a *NodeAdapter) GetNodeID() string {
	if a.node == nil {
		return "local"
	}
	return a.node.GetID()
}

// nodeClientManager 将 node.Node 适配为 scheduler.ClientManager 接口
type nodeClientManager struct {
	node *node.Node
}

// GetOnlineClients 返回所有在线客户端
func (m *nodeClientManager) GetOnlineClients() []*cluster.Client {
	clients := m.node.ListClients()

	result := make([]*cluster.Client, 0, len(clients))
	for _, info := range clients {
		client := &cluster.Client{
			ID:           info.ID,
			Name:         info.Name,
			Address:      info.Address,
			Port:         info.Port,
			Tags:         info.Tags,
			Capabilities: convertNodeCapabilitiesToCluster(info.Capabilities),
			Status:       cluster.ClientStatus(info.Status),
			LastSeen:     info.LastSeen,
			Metadata:     make(map[string]string),
			Connected:    info.Status == node.NodeStatusOnline,
		}

		// 复制 metadata
		for k, v := range info.Metadata {
			client.Metadata[k] = v
		}

		result = append(result, client)
	}

	return result
}

// GetClient 根据ID获取客户端
func (m *nodeClientManager) GetClient(clientID string) (*cluster.Client, bool) {
	info, err := m.node.GetClient(clientID)
	if err != nil {
		return nil, false
	}

	client := &cluster.Client{
		ID:           info.ID,
		Name:         info.Name,
		Address:      info.Address,
		Port:         info.Port,
		Tags:         info.Tags,
		Capabilities: convertNodeCapabilitiesToCluster(info.Capabilities),
		Status:       cluster.ClientStatus(info.Status),
		LastSeen:     info.LastSeen,
		Metadata:     make(map[string]string),
		Connected:    info.Status == node.NodeStatusOnline,
	}

	// 复制 metadata
	for k, v := range info.Metadata {
		client.Metadata[k] = v
	}

	return client, true
}

// SendCommand 向客户端发送命令
func (m *nodeClientManager) SendCommand(clientID string, command *cluster.Command) (map[string]interface{}, error) {
	// 将 cluster.Command 转换为 node.Command
	nodeCmd := &node.Command{
		ID:         command.ID,
		Type:       node.CommandType(command.Type),
		Payload:    command.Payload,
		FromNodeID: m.node.GetID(),
		ToNodeID:   clientID,
		CreatedAt:  time.Now(),
		Priority:   5, // 默认优先级
		RetryCount: 0,
		MaxRetries: 3,
	}

	// 将命令加入队列
	if err := m.node.QueueCommand(clientID, nodeCmd); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"commandId": command.ID,
		"status":    "queued",
	}, nil
}

// convertNodeCapabilitiesToCluster 转换节点能力格式
func convertNodeCapabilitiesToCluster(cap *node.NodeCapabilities) *cluster.Capabilities {
	if cap == nil {
		return nil
	}

	return &cluster.Capabilities{
		GPU:            cap.GPU,
		GPUCount:       cap.GPUCount,
		GPUName:        cap.GPUName,
		GPUMemory:      cap.GPUMemory,
		CPUCount:       cap.CPUCount,
		Memory:         cap.Memory,
		SupportsLlama:  cap.SupportsLlama,
		SupportsPython: cap.SupportsPython,
		CondaEnvs:      cap.CondaEnvs,
	}
}

// ==================== 节点管理 API ====================

// RegisterNode 适配节点注册 API
// POST /api/master/nodes/register
func (a *NodeAdapter) RegisterNode(c *gin.Context) {
	var nodeInfo node.NodeInfo
	if err := c.ShouldBindJSON(&nodeInfo); err != nil {
		a.log.Errorf("解析节点注册请求失败: %v", err)
		ValidationError(c, err)
		return
	}

	// 验证必需字段
	if nodeInfo.ID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	if nodeInfo.Address == "" {
		BadRequest(c, "节点地址不能为空")
		return
	}

	if nodeInfo.Port <= 0 || nodeInfo.Port > 65535 {
		BadRequest(c, "端口必须在1-65535范围内")
		return
	}

	// 调用 Node 的注册方法
	if err := a.node.RegisterClient(&nodeInfo); err != nil {
		a.log.Errorf("注册客户端失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "注册失败", err.Error())
		return
	}

	a.log.Infof("节点注册成功: %s (%s:%d)", nodeInfo.ID, nodeInfo.Address, nodeInfo.Port)
	Success(c, gin.H{
		"message": "节点注册成功",
		"node":    nodeInfo,
	})
}

// ListNodes 返回所有已注册的节点列表
// GET /api/master/nodes
func (a *NodeAdapter) ListNodes(c *gin.Context) {
	clients := a.node.ListClients()
	total, online, offline, busy := a.node.GetClientCount()

	// 转换为切片格式
	nodes := make([]node.NodeInfo, len(clients))
	for i, client := range clients {
		nodes[i] = *client
	}

	Success(c, gin.H{
		"nodes": nodes,
		"stats": gin.H{
			"total":   total,
			"online":  online,
			"offline": offline,
			"busy":    busy,
		},
	})
}

// GetNode 获取指定节点的详细信息
// GET /api/master/nodes/:id
func (a *NodeAdapter) GetNode(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	client, err := a.node.GetClient(nodeID)
	if err != nil {
		a.log.Errorf("获取节点失败: %v", err)
		NotFound(c, "节点")
		return
	}

	// 转换资源格式为前端期望的格式
	response := a.convertNodeToFrontendFormat(client)

	Success(c, gin.H{
		"client": response,
		"node":   response,
	})
}

// convertNodeToFrontendFormat 将NodeInfo转换为前端期望的格式
func (a *NodeAdapter) convertNodeToFrontendFormat(client *node.NodeInfo) gin.H {
	// 计算 GPU 相关信息
	var gpuMemoryTotal int64 = 0
	var gpuMemoryUsed int64 = 0
	var gpuPercent float64 = 0
	gpuInfoList := make([]gin.H, 0)

	if client.Resources != nil && len(client.Resources.GPUInfo) > 0 {
		for _, gpu := range client.Resources.GPUInfo {
			gpuMemoryTotal += gpu.TotalMemory
			gpuMemoryUsed += gpu.UsedMemory
			gpuPercent += gpu.Utilization

			gpuInfoList = append(gpuInfoList, gin.H{
				"index":         gpu.Index,
				"name":          gpu.Name,
				"vendor":        gpu.Vendor,
				"totalMemory":   gpu.TotalMemory,
				"usedMemory":    gpu.UsedMemory,
				"temperature":   gpu.Temperature,
				"utilization":   gpu.Utilization,
				"powerUsage":    gpu.PowerUsage,
				"driverVersion": gpu.DriverVersion,
			})
		}
		gpuPercent = gpuPercent / float64(len(client.Resources.GPUInfo))
	}

	// 计算CPU使用率百分比
	var cpuPercent float64 = 0
	if client.Resources != nil && client.Resources.CPUTotal > 0 {
		cpuPercent = float64(client.Resources.CPUUsed) / float64(client.Resources.CPUTotal) * 100
	}

	// 构建 capabilities 信息，处理 nil 情况
	capabilities := gin.H{
		"cpuCount":       0,
		"memory":         int64(0),
		"gpuCount":       0,
		"gpuMemory":      gpuMemoryTotal,
		"supportsLlama":  false,
		"supportsPython": false,
	}
	if client.Capabilities != nil {
		capabilities = gin.H{
			"cpuCount":       client.Capabilities.CPUCount,
			"memory":         client.Capabilities.Memory,
			"gpuCount":       client.Capabilities.GPUCount,
			"gpuMemory":      gpuMemoryTotal,
			"supportsLlama":  client.Capabilities.SupportsLlama,
			"supportsPython": client.Capabilities.SupportsPython,
		}
	}

	result := gin.H{
		"id":           client.ID,
		"name":         client.Name,
		"address":      client.Address,
		"port":         client.Port,
		"tags":         client.Tags,
		"status":       client.Status,
		"lastSeen":     client.LastSeen.Format(time.RFC3339),
		"connected":    client.Status == "online",
		"metadata":     client.Metadata,
		"capabilities": capabilities,
	}

	if client.Resources != nil {
		result["resources"] = gin.H{
			"cpuPercent":     cpuPercent,
			"memoryUsed":     client.Resources.MemoryUsed,
			"memoryTotal":    client.Resources.MemoryTotal,
			"gpuPercent":     gpuPercent,
			"gpuMemoryUsed":  gpuMemoryUsed,
			"gpuMemoryTotal": gpuMemoryTotal,
			"diskUsed":       client.Resources.DiskUsed,
			"diskTotal":      client.Resources.DiskTotal,
			"gpuInfo":        gpuInfoList,
			"rocmVersion":    client.Resources.ROCmVersion,
			"kernelVersion":  client.Resources.KernelVersion,
		}
	}

	return result
}

// UnregisterNode 注销节点
// DELETE /api/master/nodes/:id
func (a *NodeAdapter) UnregisterNode(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	if err := a.node.UnregisterClient(nodeID); err != nil {
		a.log.Errorf("注销节点失败: %v", err)
		NotFound(c, "节点")
		return
	}

	a.log.Infof("节点注销成功: %s", nodeID)
	Success(c, gin.H{
		"message": "节点注销成功",
		"nodeID":  nodeID,
	})
}

// ==================== 心跳管理 API ====================

// HandleHeartbeat 适配心跳 API
// POST /api/master/heartbeat
func (a *NodeAdapter) HandleHeartbeat(c *gin.Context) {
	var heartbeat node.HeartbeatMessage
	if err := c.ShouldBindJSON(&heartbeat); err != nil {
		a.log.Errorf("解析心跳请求失败: %v", err)
		ValidationError(c, err)
		return
	}

	// 验证必需字段
	if heartbeat.NodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	// 调用 Node 的心跳处理
	if err := a.node.HandleHeartbeat(heartbeat.NodeID, &heartbeat); err != nil {
		a.log.Errorf("处理心跳失败: %v", err)
		NotFound(c, "节点")
		return
	}

	a.log.Debugf("心跳处理成功: 节点=%s, 时间=%v", heartbeat.NodeID, heartbeat.Timestamp.Unix())

	// 广播客户端资源更新事件（通过 WebSocket/SSE）
	if a.eventCallback != nil && heartbeat.Resources != nil {
		// 获取更新后的客户端信息
		if client, err := a.node.GetClient(heartbeat.NodeID); err == nil {
			a.eventCallback("clientResourcesUpdated", gin.H{
				"clientId":  heartbeat.NodeID,
				"resources": a.convertNodeToFrontendFormat(client),
				"timestamp": time.Now().Unix(),
			})
		}
	}

	Success(c, gin.H{
		"message":   "心跳处理成功",
		"timestamp": time.Now().Unix(),
	})
}

// ==================== 命令管理 API ====================

// SendCommand 向指定节点发送命令
// POST /api/master/nodes/:id/command
func (a *NodeAdapter) SendCommand(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	var cmd node.Command
	if err := c.ShouldBindJSON(&cmd); err != nil {
		a.log.Errorf("解析命令请求失败: %v", err)
		ValidationError(c, err)
		return
	}

	// 设置命令目标节点
	cmd.ToNodeID = nodeID
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}

	// 生成命令 ID（如果未设置）
	if cmd.ID == "" {
		cmd.ID = fmt.Sprintf("cmd-%s-%d", nodeID, time.Now().UnixNano())
	}

	// 将命令加入队列
	if err := a.node.QueueCommand(nodeID, &cmd); err != nil {
		a.log.Errorf("命令发送失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "命令发送失败", err.Error())
		return
	}

	a.log.Infof("命令发送成功: 节点=%s, 类型=%s, ID=%s", nodeID, cmd.Type, cmd.ID)
	Success(c, gin.H{
		"message":   "命令发送成功",
		"nodeID":    nodeID,
		"command":   cmd.Type,
		"commandID": cmd.ID,
	})
}

// GetCommands 获取指定节点的待执行命令
// GET /api/master/nodes/:id/commands
func (a *NodeAdapter) GetCommands(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	// 从 Node 获取待执行命令
	commands := a.node.GetPendingCommands(nodeID)

	a.log.Debugf("返回待执行命令: 节点=%s, 数量=%d", nodeID, len(commands))
	Success(c, gin.H{
		"commands": commands,
	})
}

// ReportCommandResult 上报命令执行结果
// POST /api/master/command/result
func (a *NodeAdapter) ReportCommandResult(c *gin.Context) {
	var req struct {
		NodeID    string      `json:"node_id"`
		CommandID string      `json:"command_id"`
		Success   bool        `json:"success"`
		Output    string      `json:"output"`
		Error     string      `json:"error,omitempty"`
		Metadata  interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		a.log.Errorf("解析命令结果请求失败: %v", err)
		ValidationError(c, err)
		return
	}

	// 验证必需字段
	if req.NodeID == "" || req.CommandID == "" {
		BadRequest(c, "节点ID和命令ID不能为空")
		return
	}

	// 构建命令结果对象
	result := &node.CommandResult{
		CommandID:   req.CommandID,
		FromNodeID:  req.NodeID,
		ToNodeID:    a.node.GetID(),
		Success:     req.Success,
		CompletedAt: time.Now(),
	}

	// 添加结果数据
	if req.Output != "" {
		if result.Result == nil {
			result.Result = make(map[string]interface{})
		}
		result.Result["output"] = req.Output
	}

	if req.Error != "" {
		result.Error = req.Error
	}

	if req.Metadata != nil {
		// 转换 metadata 为字符串映射
		if metadataMap, ok := req.Metadata.(map[string]interface{}); ok {
			result.Metadata = make(map[string]string)
			for k, v := range metadataMap {
				result.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// 计算执行时长（毫秒）
	startTime := time.Now()

	// 存储命令结果到持久化存储
	if err := a.node.StoreCommandResult(result); err != nil {
		a.log.Errorf("存储命令结果失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "存储命令结果失败", err.Error())
		return
	}

	result.Duration = time.Since(startTime).Milliseconds()

	a.log.Infof("命令结果已存储: 节点=%s, 命令=%s, 成功=%v, 耗时=%dms",
		req.NodeID, req.CommandID, req.Success, result.Duration)

	if req.Error != "" {
		a.log.Errorf("命令执行失败: %s", req.Error)
	}

	SuccessWithMessage(c, "命令结果已记录")
}

// ==================== 路由注册 ====================

// RegisterRoutes 注册所有 API 路由（统一版本）
func (a *NodeAdapter) RegisterRoutes(router *gin.RouterGroup) {
	// ========== 主路由：/api/nodes/* ==========
	nodes := router.Group("/nodes")
	{
		// 节点管理
		nodes.POST("/register", a.RegisterNode)
		nodes.GET("", a.ListNodes)
		nodes.GET("/:id", a.GetNode)
		nodes.DELETE("/:id", a.UnregisterNode)
		nodes.POST("/:id/command", a.SendCommand)
		nodes.GET("/:id/commands", a.GetCommands)
		nodes.POST("/:id/heartbeat", a.HandleHeartbeat)
		nodes.GET("/:id/config", a.GetNodeConfig)
		nodes.POST("/:id/test", a.TestNodeLlamacpp)
	}

	// ========== 全局路由 ==========
	router.POST("/heartbeat", a.HandleHeartbeat)
	router.POST("/command/result", a.ReportCommandResult)

	// ========== 任务管理 ==========
	router.POST("/tasks", a.CreateTask)
	router.GET("/tasks", a.ListTasks)
	router.DELETE("/tasks/:id", a.DeleteTask)
	router.POST("/tasks/:id/retry", a.RetryTask)

	// ========== 网络扫描 ==========
	router.POST("/scan", a.HandleScanClients)
	router.GET("/scan/status", a.GetClientScanStatus)

	// ========== 集群概览 ==========
	router.GET("/overview", a.GetClusterOverview)

	// ========== 兼容性路由（已废弃）==========
	a.registerDeprecatedRoutes(router)

	a.log.Infof("Node API 适配器路由已注册")
}

// registerDeprecatedRoutes 注册废弃的兼容性路由
func (a *NodeAdapter) registerDeprecatedRoutes(router *gin.RouterGroup) {
	master := router.Group("/master")
	deprecatedMiddleware := a.deprecationWarningMiddleware()

	// /api/master/nodes/* -> /api/nodes/* (废弃)
	nodes := master.Group("/nodes")
	{
		nodes.Use(deprecatedMiddleware)

		nodes.POST("/register", a.RegisterNode)
		nodes.GET("", a.ListNodes)
		nodes.GET("/:id", a.GetNode)
		nodes.DELETE("/:id", a.UnregisterNode)
		nodes.POST("/:id/command", a.SendCommand)
		nodes.GET("/:id/commands", a.GetCommands)
		nodes.POST("/:id/heartbeat", a.HandleHeartbeat)
	}

	// /api/master/clients/* -> /api/nodes/* (废弃)
	clients := master.Group("/clients")
	{
		clients.Use(deprecatedMiddleware)

		clients.POST("/register", a.RegisterNode)
		clients.GET("", a.ListClients)
		clients.GET("/:id", a.GetNode)
		clients.DELETE("/:id", a.UnregisterNode)
	}

	// /api/master/heartbeat -> /api/heartbeat (废弃)
	master.POST("/heartbeat", deprecatedMiddleware, a.HandleHeartbeat)

	// /api/master/command/result -> /api/command/result (废弃)
	master.POST("/command/result", deprecatedMiddleware, a.ReportCommandResult)

	// /api/master/scan -> /api/scan (废弃)
	master.POST("/scan", deprecatedMiddleware, a.HandleScanClients)
	master.GET("/scan/status", deprecatedMiddleware, a.GetClientScanStatus)

	// /api/master/overview -> /api/overview (废弃)
	master.GET("/overview", deprecatedMiddleware, a.GetClusterOverview)

	// /api/master/tasks -> /api/tasks (废弃)
	master.GET("/tasks", deprecatedMiddleware, a.ListTasks)
	master.POST("/tasks", deprecatedMiddleware, a.CreateTask)
	master.DELETE("/tasks/:id", a.DeleteTask)
	master.POST("/tasks/:id/retry", deprecatedMiddleware, a.RetryTask)
}

// deprecationWarningMiddleware 废弃路由警告中间件
func (a *NodeAdapter) deprecationWarningMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 添加响应头警告
		c.Header("X-API-Deprecation", "This API route is deprecated. Please use /api/nodes/* instead.")
		c.Header("X-API-Sunset", "2026-12-31") // 预计废弃日期

		// 记录日志
		a.log.Warnf("使用废弃路由: %s %s", c.Request.Method, c.Request.URL.Path)

		c.Next()
	}
}

// ==================== 辅助方法 ====================

// GetNodeInstance 返回底层 Node 实例
// 供需要直接访问 Node 的场景使用
func (a *NodeAdapter) GetNodeInstance() *node.Node {
	return a.node
}

// SetNodeInstance 设置底层 Node 实例
func (a *NodeAdapter) SetNodeInstance(n *node.Node) {
	a.node = n
}

// HandleScanClients 处理客户端扫描请求
// POST /api/master/scan
func (a *NodeAdapter) HandleScanClients(c *gin.Context) {
	var req struct {
		CIDR      string `json:"cidr,omitempty"`
		PortRange string `json:"portRange,omitempty"`
		Timeout   int    `json:"timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		a.log.Errorf("解析扫描请求失败: %v", err)
	}

	go func() {
		if _, err := a.scanner.Scan(); err != nil {
			a.log.Errorf("网络扫描失败: %v", err)
		}
	}()

	SuccessWithMessage(c, "网络扫描已启动")
}

// GetClientScanStatus 获取客户端扫描状态
// GET /api/master/scan/status
func (a *NodeAdapter) GetClientScanStatus(c *gin.Context) {
	status := a.scanner.GetStatus()
	Success(c, status)
}

// ==================== 任务管理 API ====================

// ListTasks 返回所有任务列表
// GET /api/master/tasks
func (a *NodeAdapter) ListTasks(c *gin.Context) {
	tasks := a.scheduler.ListTasks()

	Success(c, gin.H{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// CreateTask 创建新任务
// POST /api/master/tasks
func (a *NodeAdapter) CreateTask(c *gin.Context) {
	var req struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		a.log.Errorf("解析任务创建请求失败: %v", err)
		ValidationError(c, err)
		return
	}

	if req.Type == "" {
		BadRequest(c, "任务类型不能为空")
		return
	}

	// 使用 scheduler 提交任务
	task, err := a.scheduler.SubmitTask(cluster.TaskType(req.Type), req.Payload)
	if err != nil {
		a.log.Errorf("创建任务失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "创建任务失败", err.Error())
		return
	}

	a.log.Infof("任务创建成功: ID=%s, Type=%s", task.ID, task.Type)
	Success(c, gin.H{
		"task": task,
	})
}

// DeleteTask 删除任务
// DELETE /api/master/tasks/:id
func (a *NodeAdapter) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		BadRequest(c, "任务ID不能为空")
		return
	}

	// 使用 scheduler 取消任务
	if err := a.scheduler.CancelTask(taskID); err != nil {
		a.log.Errorf("删除任务失败: %v", err)
		NotFound(c, "任务")
		return
	}

	a.log.Infof("任务删除成功: %s", taskID)
	Success(c, gin.H{
		"message": "任务删除成功",
		"taskId":  taskID,
	})
}

// RetryTask 重试任务
// POST /api/master/tasks/:id/retry
func (a *NodeAdapter) RetryTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		BadRequest(c, "任务ID不能为空")
		return
	}

	// 使用 scheduler 重试任务
	if err := a.scheduler.RetryTask(taskID); err != nil {
		a.log.Errorf("重试任务失败: %v", err)
		NotFound(c, "任务")
		return
	}

	// 获取更新后的任务
	task, exists := a.scheduler.GetTask(taskID)
	if !exists {
		NotFound(c, fmt.Sprintf("任务 %s", taskID))
		return
	}

	a.log.Infof("任务重试: %s", taskID)
	Success(c, gin.H{
		"message": "任务已重置为待处理状态",
		"task":    task,
	})
}

// ==================== 兼容性 API 方法 ====================

// ListClients 返回客户端列表（前端期望的格式）
// GET /api/master/clients
func (a *NodeAdapter) ListClients(c *gin.Context) {
	clients := a.node.ListClients()
	total, online, offline, busy := a.node.GetClientCount()

	// 转换为前端期望的格式
	clientList := make([]gin.H, len(clients))
	for i, client := range clients {
		// 计算 GPU 内存总量
		var gpuMemoryTotal int64 = 0
		var gpuMemoryUsed int64 = 0
		var gpuPercent float64 = 0
		gpuInfoList := make([]gin.H, 0)

		if client.Resources != nil && len(client.Resources.GPUInfo) > 0 {
			for _, gpu := range client.Resources.GPUInfo {
				gpuMemoryTotal += gpu.TotalMemory
				gpuMemoryUsed += gpu.UsedMemory
				gpuPercent += gpu.Utilization

				gpuInfoList = append(gpuInfoList, gin.H{
					"index":         gpu.Index,
					"name":          gpu.Name,
					"vendor":        gpu.Vendor,
					"totalMemory":   gpu.TotalMemory,
					"usedMemory":    gpu.UsedMemory,
					"temperature":   gpu.Temperature,
					"utilization":   gpu.Utilization,
					"powerUsage":    gpu.PowerUsage,
					"driverVersion": gpu.DriverVersion,
				})
			}
			// 平均GPU使用率
			gpuPercent = gpuPercent / float64(len(client.Resources.GPUInfo))
		}

		// 计算CPU使用率百分比
		var cpuPercent float64 = 0
		if client.Resources != nil && client.Resources.CPUTotal > 0 {
			cpuPercent = float64(client.Resources.CPUUsed) / float64(client.Resources.CPUTotal) * 100
		}

		clientList[i] = gin.H{
			"id":        client.ID,
			"name":      client.Name,
			"address":   client.Address,
			"port":      client.Port,
			"tags":      client.Tags,
			"status":    client.Status,
			"lastSeen":  client.LastSeen.Format(time.RFC3339),
			"connected": client.Status == "online",
			"metadata":  client.Metadata,
			"capabilities": gin.H{
				"cpuCount":       client.Capabilities.CPUCount,
				"memory":         client.Capabilities.Memory,
				"gpuCount":       client.Capabilities.GPUCount,
				"gpuMemory":      gpuMemoryTotal,
				"supportsLlama":  client.Capabilities.SupportsLlama,
				"supportsPython": client.Capabilities.SupportsPython,
			},
		}

		// 添加资源信息（如果存在）
		if client.Resources != nil {
			clientList[i]["resources"] = gin.H{
				"cpuPercent":     cpuPercent,
				"memoryUsed":     client.Resources.MemoryUsed,
				"memoryTotal":    client.Resources.MemoryTotal,
				"gpuPercent":     gpuPercent,
				"gpuMemoryUsed":  gpuMemoryUsed,
				"gpuMemoryTotal": gpuMemoryTotal,
				"diskUsed":       client.Resources.DiskUsed,
				"diskTotal":      client.Resources.DiskTotal,
				"gpuInfo":        gpuInfoList,
				"rocmVersion":    client.Resources.ROCmVersion,
				"kernelVersion":  client.Resources.KernelVersion,
			}
		}
	}

	Success(c, gin.H{
		"clients": clientList,
		"total":   total,
		"stats": gin.H{
			"total":   total,
			"online":  online,
			"offline": offline,
			"busy":    busy,
		},
	})
}

// GetClusterOverview 返回集群概览
// GET /api/master/overview
func (a *NodeAdapter) GetClusterOverview(c *gin.Context) {
	total, online, offline, busy := a.node.GetClientCount()

	// 从 scheduler 获取任务统计
	tasks := a.scheduler.ListTasks()
	totalTasks := len(tasks)
	runningTasks := 0
	completedTasks := 0
	failedTasks := 0
	pendingTasks := 0

	for _, task := range tasks {
		switch task.Status {
		case cluster.TaskStatusRunning:
			runningTasks++
		case cluster.TaskStatusCompleted:
			completedTasks++
		case cluster.TaskStatusFailed:
			failedTasks++
		case cluster.TaskStatusPending:
			pendingTasks++
		}
	}

	Success(c, gin.H{
		"totalClients":   total,
		"onlineClients":  online,
		"offlineClients": offline,
		"busyClients":    busy,
		"totalTasks":     totalTasks,
		"pendingTasks":   pendingTasks,
		"runningTasks":   runningTasks,
		"completedTasks": completedTasks,
		"failedTasks":    failedTasks,
		"nodes": gin.H{
			"stats": gin.H{
				"total":   total,
				"online":  online,
				"offline": offline,
				"busy":    busy,
			},
		},
	})
}

// GetNodeConfig 获取节点配置信息
// GET /api/nodes/:id/config
func (a *NodeAdapter) GetNodeConfig(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	// 向节点发送获取配置命令
	cmd := node.Command{
		ID:        fmt.Sprintf("cmd-config-%d", time.Now().UnixNano()),
		Type:      node.CommandTypeGetConfig,
		Payload:   map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	if err := a.node.QueueCommand(nodeID, &cmd); err != nil {
		a.log.Errorf("发送获取配置命令失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "发送命令失败", err.Error())
		return
	}

	a.log.Infof("获取配置命令已发送: 节点=%s", nodeID)
	Success(c, gin.H{
		"message":   "获取配置命令已发送，请通过命令结果接口查询",
		"nodeID":    nodeID,
		"commandID": cmd.ID,
	})
}

// TestNodeLlamacpp 测试节点 llama.cpp 可用性
// POST /api/nodes/:id/test
func (a *NodeAdapter) TestNodeLlamacpp(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		BadRequest(c, "节点ID不能为空")
		return
	}

	var req struct {
		BinaryPath string `json:"binary_path,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 忽略错误，因为是可选参数
	}

	// 向节点发送测试命令
	cmd := node.Command{
		ID:        fmt.Sprintf("cmd-test-%d", time.Now().UnixNano()),
		Type:      node.CommandTypeTestLlamacpp,
		Payload:   map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	if req.BinaryPath != "" {
		cmd.Payload["binary_path"] = req.BinaryPath
	}

	if err := a.node.QueueCommand(nodeID, &cmd); err != nil {
		a.log.Errorf("发送测试命令失败: %v", err)
		ErrorWithDetails(c, types.ErrInternalError, "发送命令失败", err.Error())
		return
	}

	a.log.Infof("测试 llama.cpp 命令已发送: 节点=%s", nodeID)
	Success(c, gin.H{
		"message":   "测试命令已发送，请通过命令结果接口查询",
		"nodeID":    nodeID,
		"commandID": cmd.ID,
	})
}
