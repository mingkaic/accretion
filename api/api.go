package api

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/mingkaic/accretion/proto"
	"github.com/mingkaic/accretion/service"
)

type (
	AccretionAPI interface {
		Run(string, []grpc.ServerOption) error
	}

	accretionAPI struct {
		server proto.TenncorProfileServiceServer
	}

	tenncorProfileServiceServer struct {
		proto.UnimplementedTenncorProfileServiceServer
	}
)

func NewTenncorProfileService() proto.TenncorProfileServiceServer {
	return &tenncorProfileServiceServer{}
}

func (tenncorProfileServiceServer) ListProfile(
	ctx context.Context, req *proto.ListProfileRequest) (
	*proto.ListProfileResponse, error) {
	return &proto.ListProfileResponse{}, nil
}

func (tenncorProfileServiceServer) CreateProfile(
	ctx context.Context, req *proto.CreateProfileRequest) (
	*proto.CreateProfileResponse, error) {
	log.Debug("creating profile")
	svc := service.NewGraphService()
	if err := svc.CreateGraphProfile(req.Model, req.Runtime); err != nil {
		log.Debugf("failed profile creation: %v", err)
		return nil, err
	}
	log.Debug("created profile")
	return &proto.CreateProfileResponse{}, nil
}

func NewAccretionAPI() AccretionAPI {
	out := &accretionAPI{
		server: NewTenncorProfileService(),
	}
	return out
}

func (a *accretionAPI) Run(addr string, opts []grpc.ServerOption) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterTenncorProfileServiceServer(grpcServer, a.server)
	grpcServer.Serve(listener)
	return nil
}
