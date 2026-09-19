package cluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

type NodePool interface {
	// Upsert updates or inserts a node into the pool.
	Upsert(ctx context.Context, node *ent.Node)
	// Get returns a node with the given capability and preferred node id. `allowed` is a list of allowed node ids.
	// If `allowed` is empty, all nodes with the capability are considered.
	Get(ctx context.Context, capability types.NodeCapability, preferred int, allowed []int) (Node, error)
}

type (
	weightedNodePool struct {
		lock sync.RWMutex

		conf     conf.ConfigProvider
		settings setting.Provider

		nodes map[types.NodeCapability][]*nodeItem
	}

	nodeItem struct {
		node    Node
		weight  int
		current int
	}
)

var (
	ErrNoAvailableNode = fmt.Errorf("no available node found")

	supportedCapabilities = []types.NodeCapability{
		types.NodeCapabilityNone,
		types.NodeCapabilityCreateArchive,
		types.NodeCapabilityExtractArchive,
		types.NodeCapabilityRemoteDownload,
	}
)

func NewNodePool(ctx context.Context, l logging.Logger, config conf.ConfigProvider, settings setting.Provider,
	client inventory.NodeClient) (NodePool, error) {
	nodes, err := client.ListActiveNodes(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list active nodes: %w", err)
	}

	pool := &weightedNodePool{
		nodes:    make(map[types.NodeCapability][]*nodeItem),
		conf:     config,
		settings: settings,
	}
	for _, node := range nodes {
		for _, capability := range supportedCapabilities {
			// If current capability is enabled, add it to pool slot.
			if capability == types.NodeCapabilityNone ||
				(node.Capabilities != nil && node.Capabilities.Enabled(int(capability))) {
				if _, ok := pool.nodes[capability]; !ok {
					pool.nodes[capability] = make([]*nodeItem, 0)
				}

				l.Debug("Add node %q to capability slot %d with weight %d", node.Name, capability, node.Weight)
				pool.nodes[capability] = append(pool.nodes[capability], &nodeItem{
					node:    newNode(ctx, node, config, settings),
					weight:  node.Weight,
					current: 0,
				})
			}
		}
	}

	return pool, nil
}

func (p *weightedNodePool) Get(ctx context.Context, capability types.NodeCapability, preferred int, allowed []int) (Node, error) {
	l := logging.FromContext(ctx)
	p.lock.Lock()
	defer p.lock.Unlock()

	nodes, ok := p.nodes[capability]
	if !ok || len(nodes) == 0 {
		return nil, fmt.Errorf("no node found with capability %d: %w", capability, ErrNoAvailableNode)
	}

	if len(allowed) > 0 {
		allowedSet := lo.SliceToMap(allowed, func(id int) (int, struct{}) { return id, struct{}{} })
		filtered := lo.Filter(nodes, func(item *nodeItem, _ int) bool {
			_, ok := allowedSet[item.node.ID()]
			return ok
		})
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no allowed node found with capability %d: %w", capability, ErrNoAvailableNode)
		}
		nodes = filtered
	}

	var selected *nodeItem

	if preferred > 0 {
		// First try to find the preferred node.
		for _, n := range nodes {
			if n.node.ID() == preferred {
				selected = n
				break
			}
		}

		if selected == nil {
			l.Debug("Preferred node %d not found, fallback to select a node with the least current weight", preferred)
		}
	}

	if selected == nil {
		// If no preferred one, or the preferred one is not available, select a node with the least current weight.

		// Total weight of all items.
		var total int

		// Loop through the list of items and add the item's weight to the current weight.
		// Also increment the total weight counter.
		var maxNode *nodeItem
		for _, item := range nodes {
			item.current += max(1, item.weight)
			total += max(1, item.weight)

			// Select the item with max weight.
			if maxNode == nil || item.current > maxNode.current {
				maxNode = item
			}
		}

		// Select the item with the max weight.
		selected = maxNode
		if selected == nil {
			return nil, fmt.Errorf("no node found with capability %d: %w", capability, ErrNoAvailableNode)
		}

		l.Debug("Selected node %q with weight=%d, current=%d, total=%d", selected.node.Name(), selected.weight, maxNode.current, total)

		// Reduce the current weight of the selected item by the total weight.
		maxNode.current -= total
	}

	return selected.node, nil
}

func (p *weightedNodePool) Upsert(ctx context.Context, n *ent.Node) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, capability := range supportedCapabilities {
		_, index, found := lo.FindIndexOf(p.nodes[capability], func(i *nodeItem) bool {
			return i.node.ID() == n.ID
		})
		if capability == types.NodeCapabilityNone ||
			(n.Capabilities != nil && n.Capabilities.Enabled(int(capability))) {
			if n.Status != node.StatusActive && found {
				// Remove inactive node
				p.nodes[capability] = append(p.nodes[capability][:index], p.nodes[capability][index+1:]...)
				continue
			}

			if found {
				p.nodes[capability][index].node = newNode(ctx, n, p.conf, p.settings)
			} else {
				p.nodes[capability] = append(p.nodes[capability], &nodeItem{
					node:    newNode(ctx, n, p.conf, p.settings),
					weight:  n.Weight,
					current: 0,
				})
			}
		} else if found {
			// Capability changed, remove the old node.
			p.nodes[capability] = append(p.nodes[capability][:index], p.nodes[capability][index+1:]...)
		}
	}
}

type slaveDummyNodePool struct {
	conf       conf.ConfigProvider
	settings   setting.Provider
	masterNode Node
}

func NewSlaveDummyNodePool(ctx context.Context, config conf.ConfigProvider, settings setting.Provider) NodePool {
	return &slaveDummyNodePool{
		conf:     config,
		settings: settings,
		masterNode: newNode(ctx, &ent.Node{
			ID:   0,
			Name: "Master",
			Type: node.TypeMaster,
		}, config, settings),
	}
}

func (s *slaveDummyNodePool) Upsert(ctx context.Context, node *ent.Node) {
}

func (s *slaveDummyNodePool) Get(ctx context.Context, capability types.NodeCapability, preferred int, allowed []int) (Node, error) {
	return s.masterNode, nil
}
