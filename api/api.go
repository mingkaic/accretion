package api

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/mingkaic/accretion/proto/profile"
	"github.com/mingkaic/accretion/service"
)

type (
	AccretionAPI interface {
		Run(string, []grpc.ServerOption) error
	}

	accretionAPI struct {
		server profile.TenncorProfileServiceServer
	}

	tenncorProfileServiceServer struct {
		profile.UnimplementedTenncorProfileServiceServer
	}
)

func NewTenncorProfileService() profile.TenncorProfileServiceServer {
	return &tenncorProfileServiceServer{}
}

func (tenncorProfileServiceServer) ListProfile(
	ctx context.Context, req *profile.ListProfileRequest) (
	*profile.ListProfileResponse, error) {
	return &profile.ListProfileResponse{}, nil
}

func (tenncorProfileServiceServer) CreateProfile(
	ctx context.Context, req *profile.CreateProfileRequest) (
	*profile.CreateProfileResponse, error) {
	log.Debug("creating profile")
	svc := service.NewGraphService()
	if err := svc.CreateGraphProfile(req.Model, req.OperatorData); err != nil {
		log.Debugf("failed profile creation: %v", err)
		return nil, err
	}
	log.Debug("created profile")
	return &profile.CreateProfileResponse{}, nil
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
	profile.RegisterTenncorProfileServiceServer(grpcServer, a.server)
	grpcServer.Serve(listener)
	return nil
}
