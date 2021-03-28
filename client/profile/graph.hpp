#include "egrpc/egrpc.hpp"

#include "tenncor/serial.hpp"

#include "proto/profile.grpc.pb.h"

#include "client/profile/device.hpp"

#ifndef DBG_PROFILE_GRAPH_HPP
#define DBG_PROFILE_GRAPH_HPP

namespace dbg
{

namespace profile
{

struct TenncorProfileClient final : public egrpc::GrpcClient
{
	TenncorProfileClient (std::shared_ptr<grpc::ChannelInterface> channel,
		egrpc::ClientConfig cfg) : GrpcClient(cfg),
		stub_(tenncor_profile::TenncorProfileService::NewStub(channel)) {}

	grpc::Status create_profile (const tenncor_profile::CreateProfileRequest& req)
	{
		tenncor_profile::CreateProfileResponse res;
		grpc::ClientContext ctx;
		build_ctx(ctx, true);
		return stub_->CreateProfile(&ctx, req, &res);
	}

	std::unique_ptr<tenncor_profile::TenncorProfileService::Stub> stub_;
};

void remote_profile (const std::string& addr, eteq::ETensorsT roots);

}

}

#endif // DBG_PROFILE_GRAPH_HPP
