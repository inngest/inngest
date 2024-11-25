package connect

import "sync"

type hostsManager struct {
	gatewayHosts            []string
	availableGatewayHosts   map[string]struct{}
	drainingGatewayHosts    map[string]struct{}
	unreachableGatewayHosts map[string]struct{}
	hostsLock               sync.RWMutex
}

func newHostsManager(gatewayHosts []string) *hostsManager {
	hm := &hostsManager{
		gatewayHosts:            gatewayHosts,
		availableGatewayHosts:   make(map[string]struct{}),
		drainingGatewayHosts:    make(map[string]struct{}),
		unreachableGatewayHosts: make(map[string]struct{}),
	}

	hm.resetGateways()

	return hm
}

func (h *hostsManager) pickAvailableGateway() string {
	h.hostsLock.RLock()
	defer h.hostsLock.RUnlock()

	for host := range h.availableGatewayHosts {
		return host
	}
	return ""
}

func (h *hostsManager) markDrainingGateway(host string) {
	h.hostsLock.Lock()
	defer h.hostsLock.Unlock()
	delete(h.availableGatewayHosts, host)
	h.drainingGatewayHosts[host] = struct{}{}
}

func (h *hostsManager) markUnreachableGateway(host string) {
	h.hostsLock.Lock()
	defer h.hostsLock.Unlock()
	delete(h.availableGatewayHosts, host)
	h.unreachableGatewayHosts[host] = struct{}{}
}

func (h *hostsManager) resetGateways() {
	h.hostsLock.Lock()
	defer h.hostsLock.Unlock()
	h.availableGatewayHosts = make(map[string]struct{})
	h.drainingGatewayHosts = make(map[string]struct{})
	h.unreachableGatewayHosts = make(map[string]struct{})
	for _, host := range h.gatewayHosts {
		h.availableGatewayHosts[host] = struct{}{}
	}
}
