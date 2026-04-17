package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNode(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-1",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)
	require.NotNil(t, node)

	assert.Equal(t, "test-node-1", node.GetID())
	assert.Equal(t, "Test Node", node.GetName())
	assert.Equal(t, NodeRoleHybrid, node.GetRole())
	assert.Equal(t, NodeStatusOffline, node.GetStatus())
	assert.Equal(t, "localhost", node.GetAddress())
	assert.Equal(t, 8080, node.GetPort())
	assert.False(t, node.IsRunning())
}

func TestNewNodeWithNilConfig(t *testing.T) {
	node, err := NewNode(nil)
	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "配置不能为空")
}

func TestNewNodeWithEmptyID(t *testing.T) {
	config := &NodeConfig{
		ID:      "",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "节点ID不能为空")
}

func TestNodeStartStop(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-2",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	assert.False(t, node.IsRunning())
	assert.Equal(t, NodeStatusOffline, node.GetStatus())

	err = node.Start()
	require.NoError(t, err)
	assert.True(t, node.IsRunning())
	assert.Equal(t, NodeStatusOnline, node.GetStatus())
	assert.NotNil(t, node.startedAt)

	err = node.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "节点已经在运行")

	// Stop resource monitor first to avoid deadlock (ResourceMonitor callback
	// acquires Node.mu while Node.Stop() holds it waiting for ResourceMonitor)
	if node.resource != nil {
		node.resource.Stop()
	}

	err = node.Stop()
	require.NoError(t, err)
	assert.False(t, node.IsRunning())
	assert.Equal(t, NodeStatusOffline, node.GetStatus())
	assert.NotNil(t, node.stoppedAt)

	err = node.Stop()
	assert.NoError(t, err)
}

func TestNodeTags(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-3",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	tags := node.GetTags()
	assert.Empty(t, tags)

	node.AddTag("gpu")
	node.AddTag("cuda")
	node.AddTag("production")

	tags = node.GetTags()
	assert.Equal(t, []string{"gpu", "cuda", "production"}, tags)

	node.AddTag("gpu")
	tags = node.GetTags()
	assert.Equal(t, []string{"gpu", "cuda", "production"}, tags)

	node.RemoveTag("cuda")
	tags = node.GetTags()
	assert.Equal(t, []string{"gpu", "production"}, tags)

	node.RemoveTag("nonexistent")
	tags = node.GetTags()
	assert.Equal(t, []string{"gpu", "production"}, tags)

	newTags := []string{"test", "development"}
	node.SetTags(newTags)
	tags = node.GetTags()
	assert.Equal(t, newTags, tags)
}

func TestNodeMetadata(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-4",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	metadata := node.GetMetadata()
	assert.Empty(t, metadata)

	newMetadata := map[string]string{
		"region":   "us-west",
		"zone":     "1a",
		"instance": "t3.large",
	}
	node.SetMetadata(newMetadata)

	metadata = node.GetMetadata()
	assert.Equal(t, newMetadata, metadata)

	metadata["new-key"] = "new-value"
	originalMetadata := node.GetMetadata()
	assert.NotContains(t, originalMetadata, "new-key")
}

func TestNodeRole(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-5",
		Name:    "Test Node",
		Role:    NodeRoleClient,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)
	assert.Equal(t, NodeRoleClient, node.GetRole())

	err = node.SetRole(NodeRoleMaster)
	require.NoError(t, err)
	assert.Equal(t, NodeRoleMaster, node.GetRole())

	err = node.Start()
	require.NoError(t, err)

	err = node.SetRole(NodeRoleHybrid)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "节点运行时不能更改角色")
	assert.Equal(t, NodeRoleMaster, node.GetRole())

	if node.resource != nil {
		node.resource.Stop()
	}
}

func TestNodeUptime(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-6",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	uptime := node.GetUptime()
	assert.Equal(t, time.Duration(0), uptime)

	beforeStart := time.Now()
	err = node.Start()
	require.NoError(t, err)

	uptime = node.GetUptime()
	assert.Greater(t, uptime, time.Duration(0))
	assert.Less(t, uptime, time.Second)

	time.Sleep(100 * time.Millisecond)

	newUptime := node.GetUptime()
	assert.Greater(t, newUptime, uptime)

	beforeStop := time.Now()
	if node.resource != nil {
		node.resource.Stop()
	}
	err = node.Stop()
	require.NoError(t, err)

	uptimeAfterStop := node.GetUptime()
	expectedUptime := beforeStop.Sub(beforeStart)
	assert.InDelta(t, expectedUptime, uptimeAfterStop, float64(100*time.Millisecond))
}

func TestNodeToString(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-7",
		Name:    "Test Node",
		Role:    NodeRoleHybrid,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	str := node.String()
	assert.Contains(t, str, "test-node-7")
	assert.Contains(t, str, "Test Node")
	assert.Contains(t, str, "hybrid")
	assert.Contains(t, str, "offline")
	assert.Contains(t, str, "localhost:8080")
}

func TestNodeToInfo(t *testing.T) {
	config := &NodeConfig{
		ID:      "test-node-8",
		Name:    "Test Node",
		Role:    NodeRoleMaster,
		Address: "localhost",
		Port:    8080,
	}

	node, err := NewNode(config)
	require.NoError(t, err)

	node.AddTag("master")
	node.AddTag("primary")
	node.SetMetadata(map[string]string{
		"datacenter": "dc1",
		"rack":       "r1",
	})

	info := node.ToInfo()
	require.NotNil(t, info)

	assert.Equal(t, "test-node-8", info.ID)
	assert.Equal(t, "Test Node", info.Name)
	assert.Equal(t, NodeRoleMaster, info.Role)
	assert.Equal(t, NodeStatusOffline, info.Status)
	assert.Equal(t, "localhost", info.Address)
	assert.Equal(t, 8080, info.Port)
	assert.Equal(t, []string{"master", "primary"}, info.Tags)
	assert.Equal(t, map[string]string{
		"datacenter": "dc1",
		"rack":       "r1",
	}, info.Metadata)

	assert.NotNil(t, info.Capabilities)
	assert.NotNil(t, info.Resources)

	info.Name = "Modified Name"
	assert.Equal(t, "Test Node", node.GetName())
}
