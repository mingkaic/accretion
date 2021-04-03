package api

import (
	"context"
	"net"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/zenazn/goji/graceful"
	"google.golang.org/grpc"

	"github.com/mingkaic/accretion/proto/profile"
	"github.com/mingkaic/accretion/service"
)

type (
	AccretionAPI interface {
		Run(string, string, chan error, []grpc.ServerOption, HTTPOpts)
	}

	HTTPOpts struct {
		MuxOpts  []runtime.ServeMuxOption
		DialOpts []grpc.DialOption
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
	log.Debug("listing profiles")
	svc := service.NewGraphService()
	profiles, err := svc.ListGraphProfiles()
	if err != nil {
		return nil, err
	}
	return &profile.ListProfileResponse{
		Profiles: profiles,
	}, nil
}

func (tenncorProfileServiceServer) GetProfile(
	ctx context.Context, req *profile.GetProfileRequest) (
	*profile.GetProfileResponse, error) {
	profileId := req.GetProfileId()
	log.Debug("getting profile %s", profileId)
	svc := service.NewGraphService()
	nodes, edges, err := svc.GetGraphProfile(profileId)
	if err != nil {
		return nil, err
	}
	return &profile.GetProfileResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (tenncorProfileServiceServer) CreateProfile(
	ctx context.Context, req *profile.CreateProfileRequest) (
	*profile.CreateProfileResponse, error) {
	id := uuid.NewString()
	log.Debugf("creating profile %s", id)
	svc := service.NewGraphService()
	if err := svc.CreateGraphProfile(id, req.Model, req.OperatorData); err != nil {
		log.Debugf("failed profile %s creation: %v", id, err)
		return nil, err
	}
	log.Debugf("created profile %s", id)
	return &profile.CreateProfileResponse{
		ProfileId: id,
	}, nil
}

func NewAccretionAPI() AccretionAPI {
	out := &accretionAPI{
		server: NewTenncorProfileService(),
	}
	return out
}

func (a *accretionAPI) Run(httpAddr, grpcAddr string, errs chan error, grpcOpts []grpc.ServerOption, httpOpts HTTPOpts) {
	go func() {
		errs <- a.runGRPC(grpcAddr, grpcOpts)
	}()
	go func() {
		errs <- a.runHTTP(httpAddr, grpcAddr, httpOpts.MuxOpts, httpOpts.DialOpts)
	}()
}

func (a *accretionAPI) runGRPC(addr string, opts []grpc.ServerOption) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(opts...)
	profile.RegisterTenncorProfileServiceServer(grpcServer, a.server)
	return graceful.Serve(listener, grpcServer)
}

func (a *accretionAPI) runHTTP(httpAddr, grpcAddr string, muxOpts []runtime.ServeMuxOption, dialOpts []grpc.DialOption) error {
	listener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(muxOpts...)
	err = profile.RegisterTenncorProfileServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts)
	if err != nil {
		return err
	}
	return graceful.Serve(listener, mux)
}
