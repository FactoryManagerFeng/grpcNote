package weight

import (
	"context"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"math/rand"
	"sync"
)

// Name is the name of round_robin balancer.
const Name = "weight"

// newBuilder creates a new roundrobin balancer builder.
func newBuilder() balancer.Builder {
	return base.NewBalancerBuilderWithConfig(Name, &rrPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

var (
	minWeight = 1
	maxWeight = 5
)

type attributeKey struct{}

// 权重额外信息结构
type AddrInfo struct {
	Weight int
}

// 设置权重信息在地址中
func SetAddrInfo(addr resolver.Address, addrInfo AddrInfo) resolver.Address {
	addr.Metadata = addrInfo
	return addr
}

// 获取地址中的权重信息
func GetAddrInfo(addr resolver.Address) AddrInfo {
	v := addr.Metadata
	ai, _ := v.(AddrInfo)
	return ai
}

type rrPickerBuilder struct{}

// 加权随机法，如果一个地址权重为5，那就生成5个subconn，如果权重是1就生成1个subconn，
// pick的时候，采用随机算法，这样就相当于按照权重的方式来调用对应的地址了
func (*rrPickerBuilder) Build(readySCs map[resolver.Address]balancer.SubConn) balancer.Picker {
	grpclog.Infof("roundrobinPicker: newPicker called with readySCs: %v", readySCs)
	var scs []balancer.SubConn
	for addr, sc := range readySCs {

		node := GetAddrInfo(addr)
		nodeWeight := node.Weight
		// 设置权重不能超过最大最小值范围
		if nodeWeight < minWeight {
			nodeWeight = minWeight
		} else if nodeWeight > maxWeight {
			nodeWeight = maxWeight
		}
		for i := 0; i < nodeWeight; i++ {
			scs = append(scs, sc)
		}
	}
	return &rrPicker{
		subConns: scs,
	}
}

type rrPicker struct {
	subConns []balancer.SubConn

	mu   sync.Mutex
	next int
}

func (p *rrPicker) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	if len(p.subConns) <= 0 {
		return nil, nil, balancer.ErrNoSubConnAvailable
	}

	p.mu.Lock()
	// 采用随机算法，随机获取subConn
	index := rand.Intn(len(p.subConns))
	sc := p.subConns[index]
	p.mu.Unlock()
	return sc, nil, nil
}
