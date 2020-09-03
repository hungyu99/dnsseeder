package main

import (
	"context"
	"fmt"
	"net"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/infrastructure/network/dnsseed/pb"
	"github.com/kaspanet/kaspad/util/subnetworkid"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type GRPCServer interface {
	Start(listenInterface string) error
	Stop()
}

type grpcServer struct {
	pb.UnimplementedPeerServiceServer

	server *grpc.Server
	amgr   *Manager
}

func NewGRPCServer(amgr *Manager) GRPCServer {
	return &grpcServer{amgr: amgr}
}

func (s *grpcServer) Start(listenInterface string) error {
	s.server = grpc.NewServer()
	pb.RegisterPeerServiceServer(s.server, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(listenInterface))
	if err != nil {
		return errors.WithStack(err)
	}

	spawn("gRPC server", func() {
		err = s.server.Serve(lis)
		if err != nil {
			fmt.Printf("%+v", err)
		}
	})

	return nil
}

func (s *grpcServer) Stop() {
	s.server.Stop()
	s.server = nil
}

func (s *grpcServer) GetPeersList(ctx context.Context, req *pb.GetPeersListRequest) (*pb.GetPeersListResponse, error) {

	subnetworkID, err := FromProtobufSubnetworkID(req.SubnetworkID)

	if err != nil {
		return nil, err
	}

	// mb, we should move DNS-related logic out of manager?
	ipv4Addresses := s.amgr.GoodAddresses(dns.TypeA, appmessage.ServiceFlag(req.ServiceFlag), req.IncludeAllSubnetworks, subnetworkID)
	ipv6Addresses := s.amgr.GoodAddresses(dns.TypeAAAA, appmessage.ServiceFlag(req.ServiceFlag), req.IncludeAllSubnetworks, subnetworkID)

	addresses := ToProtobufAddresses(append(ipv4Addresses, ipv6Addresses...))
	log.Errorf("ADDRESSES: %+v", addresses)

	return &pb.GetPeersListResponse{Addresses: addresses}, nil
}

func FromProtobufSubnetworkID(proto []byte) (*subnetworkid.SubnetworkID, error) {
	if len(proto) == 0 {
		return nil, nil
	}

	subnetworkID, err := subnetworkid.New(proto)
	if err != nil {
		return nil, err
	}

	return subnetworkID, nil
}

func ToProtobufAddresses(addresses []*appmessage.NetAddress) []*pb.NetAddress {
	var protoAddresses []*pb.NetAddress

	for _, addr := range addresses {
		proto := &pb.NetAddress{
			Timestamp: addr.Timestamp.UnixSeconds(),
			Services:  uint64(addr.Services),
			IP:        []byte(addr.IP),
			Port:      uint32(addr.Port),
		}
		protoAddresses = append(protoAddresses, proto)
	}

	return protoAddresses
}
