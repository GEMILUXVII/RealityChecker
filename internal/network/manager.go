package network

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"RealityChecker/internal/types"
)

// ConnectionManager 连接管理器
// 每次按需创建新的 TLS 连接（确保 ALPN 协商与 SNI 正确），不做连接复用。
type ConnectionManager struct {
	config *types.Config
	mu     sync.RWMutex
	stats  *types.ConnectionStats
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(config *types.Config) *ConnectionManager {
	return &ConnectionManager{
		config: config,
		stats:  &types.ConnectionStats{},
	}
}

// Start 启动连接管理器
func (cm *ConnectionManager) Start() error {
	return nil
}

// Stop 停止连接管理器
func (cm *ConnectionManager) Stop() error {
	return nil
}

// GetTLSConnection 获取TLS连接
func (cm *ConnectionManager) GetTLSConnection(ctx context.Context, domain string) (*tls.Conn, error) {
	// 总是创建新的TLS连接，确保ALPN协商正确
	const tlsPort = ":443"
	tcpConn, err := net.DialTimeout("tcp", domain+tlsPort, cm.config.Network.Timeout)
	if err != nil {
		cm.mu.Lock()
		cm.stats.FailedConnections++
		cm.mu.Unlock()
		return nil, err
	}

	// 创建TLS连接
	// 关闭握手层的证书校验：证书有效性由综合TLS检测器手动验证（链/有效期/主机名），
	// 以便对"证书不可信/过期/SNI不匹配"等情况给出准确原因，而不是一律表现为握手失败。
	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName:         domain,
		NextProtos:         []string{"h2", "http/1.1"}, // h2优先
		InsecureSkipVerify: true,
	})

	// 执行TLS握手
	if err := tlsConn.Handshake(); err != nil {
		tcpConn.Close()
		cm.mu.Lock()
		cm.stats.FailedConnections++
		cm.mu.Unlock()
		return nil, err
	}

	cm.mu.Lock()
	cm.stats.TotalConnections++
	cm.stats.ActiveConnections++
	cm.mu.Unlock()
	return tlsConn, nil
}

// CloseTLSConnection 关闭TLS连接
func (cm *ConnectionManager) CloseTLSConnection(conn *tls.Conn) {
	if conn != nil {
		conn.Close()
		cm.mu.Lock()
		cm.stats.ActiveConnections--
		cm.mu.Unlock()
	}
}

// GetStats 获取连接统计信息
func (cm *ConnectionManager) GetStats() *types.ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return &types.ConnectionStats{
		ActiveConnections: cm.stats.ActiveConnections,
		TotalConnections:  cm.stats.TotalConnections,
		FailedConnections: cm.stats.FailedConnections,
	}
}
