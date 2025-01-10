package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCServer struct {
	etcdClient  *clientv3.Client
	serviceName string
	serviceAddr string
	grpcServer  *grpc.Server

	// 服务发现相关
	servers map[string]string           // 服务列表
	clients map[string]*grpc.ClientConn // 到其他服务的连接
	lock    sync.RWMutex
	stopCh  chan struct{}
}

// 创建服务器实例
func NewGRPCServer(serviceName, serviceAddr string) (*GRPCServer, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create etcd client failed: %v", err)
	}
	return &GRPCServer{
		etcdClient:  etcdClient,
		serviceName: serviceName,
		serviceAddr: serviceAddr,
		servers:     make(map[string]string),
		clients:     make(map[string]*grpc.ClientConn),
		stopCh:      make(chan struct{}),
	}, nil
}

// 服务注册
func (s *GRPCServer) Register() error {
	ctx := context.Background()
	// 创建租约
	lease, err := s.etcdClient.Grant(ctx, 5)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}

	// 注册服务
	key := fmt.Sprintf("/services/%s/%s", s.serviceName, s.serviceAddr)
	_, err = s.etcdClient.Put(ctx, key, s.serviceAddr, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("put service failed: %v", err)
	}

	// 保持租约
	keepAliveCh, err := s.etcdClient.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("keep alive failed: %v", err)
	}

	go func() {
		for {
			select {
			case <-keepAliveCh:
				// 续约成功
			case <-s.stopCh:
				return
			}
		}
	}()

	return nil
}

// 服务发现
func (s *GRPCServer) WatchServices() error {
	prefix := "/services/"

	// 获取现有服务
	resp, err := s.etcdClient.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		s.addService(string(kv.Key), string(kv.Value))
	}

	// 监听服务变更
	go s.watchServices(prefix)
	return nil
}

// 监听服务变更
func (s *GRPCServer) watchServices(prefix string) {
	watchCh := s.etcdClient.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for {
		select {
		case resp := <-watchCh:
			for _, ev := range resp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					s.addService(string(ev.Kv.Key), string(ev.Kv.Value))
				case clientv3.EventTypeDelete:
					s.removeService(string(ev.Kv.Key))
				}
			}
		case <-s.stopCh:
			return
		}
	}
}

// 添加服务
func (s *GRPCServer) addService(key, addr string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// 跳过自己的地址
	if addr == s.serviceAddr {
		return nil
	}

	// 检查是否需要更新
	if oldAddr, exists := s.servers[key]; exists {
		if oldAddr == addr {
			return nil
		}
		if conn, ok := s.clients[key]; ok {
			conn.Close()
			delete(s.clients, key)
		}
	}

	// 建立新连接
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	s.servers[key] = addr
	s.clients[key] = conn
	log.Printf("Added service: %s -> %s", key, addr)
	return nil
}

// 移除服务
func (s *GRPCServer) removeService(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if conn, ok := s.clients[key]; ok {
		conn.Close()
		delete(s.clients, key)
	}
	delete(s.servers, key)
	log.Printf("Removed service: %s", key)
}

// 获取服务列表
func (s *GRPCServer) GetServices() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	services := make(map[string]string)
	for k, v := range s.servers {
		services[k] = v
	}
	return services
}

// 获取服务客户端
func (s *GRPCServer) GetClient(name string) (*grpc.ClientConn, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for sname, conn := range s.clients {
		if sname == name {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("no available services")
}

// 启动服务器
func (s *GRPCServer) StartGRPC(registerRpcServer func(*grpc.Server)) error {
	// 注册服务
	if err := s.Register(); err != nil {
		return fmt.Errorf("register service failed: %v", err)
	}

	// 启动服务发现
	if err := s.WatchServices(); err != nil {
		return fmt.Errorf("watch services failed: %v", err)
	}

	// 创建 gRPC 服务器
	lis, err := net.Listen("tcp", s.serviceAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s.grpcServer = grpc.NewServer()
	registerRpcServer(s.grpcServer)

	log.Printf("Server starting at %s", s.serviceAddr)
	return s.grpcServer.Serve(lis)
}

// 停止服务器
func (s *GRPCServer) Stop() {
	close(s.stopCh)

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	for _, conn := range s.clients {
		conn.Close()
	}

	if s.etcdClient != nil {
		s.etcdClient.Close()
	}
}
