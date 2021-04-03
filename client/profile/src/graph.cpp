#include "client/profile/graph.hpp"

#ifdef DBG_PROFILE_GRAPH_HPP

namespace dbg
{

namespace profile
{

void remote_profile (const std::string& addr, eteq::ETensorsT roots)
{
    global::infof("profiling to remote address %s", addr.c_str());
    auto channel = grpc::CreateChannel(addr,
        grpc::InsecureChannelCredentials());
    TenncorProfileClient client(channel, egrpc::ClientConfig{
        std::chrono::milliseconds(50000),
        std::chrono::milliseconds(100000),
        3,
    });

    tenncor_profile::CreateProfileRequest req;
    onnx::ModelProto* pb_model = req.mutable_model();

    eigen::Device realdev(
        eigen::get_runtime(), std::numeric_limits<size_t>::max());
    ProfilerDevice device(realdev);

    teq::TensSetT rootset;
    teq::multi_get(roots.begin(), roots.end(),
        std::inserter(rootset, rootset.end()));
    teq::Evaluator().evaluate(device, rootset);
    auto owners = teq::track_ownptrs(roots);

    auto gen = global::get_generator();
    onnx::TensptrIdT ids;
    auto op_data = req.mutable_operator_data();
    for (auto stat : device.stats_)
    {
        auto key = stat.first;
        auto val = stat.second;
        auto uuid = gen->get_str();

        tenncor_profile::FuncInfo finfo;
		teq::Shape shape = key->shape();
		auto shapel = shape.to_list();
        serial::marshal_tensor(
        [&finfo,uuid,shapel]() -> onnx::TensorProto*
        {
            auto pb_tens = finfo.mutable_dense_data();
            pb_tens->set_name(uuid);
            google::protobuf::RepeatedField<int64_t> slist(
                shapel.begin(), shapel.end());
            pb_tens->mutable_dims()->Swap(&slist);
            return pb_tens;
        },
        [&finfo,uuid,shapel]() -> onnx::SparseTensorProto*
        {
            auto pb_stens = finfo.mutable_sparse_data();
            auto pb_tens = pb_stens->mutable_values();
            pb_tens->set_name(uuid);
            google::protobuf::RepeatedField<int64_t> slist(
                shapel.begin(), shapel.end());
            pb_tens->mutable_dims()->Swap(&slist);
            return pb_stens;
        }, *key);
        finfo.set_runtime((uint64_t) val);

        op_data->insert({uuid, finfo});
        ids.insert({owners.at(key), uuid});
    }
    tcr::save_model(*pb_model, roots, ids);

    tenncor_profile::CreateProfileResponse res;
    grpc::Status status = client.create_profile(req, res);
    if (status.ok())
    {
        auto id = res.profile_id();
        global::infof("successfully created profile %s in `%s`",
            id.c_str(), addr.c_str());
    }
    else
    {
        global::errorf("failed to create profile in `%s`: %s",
            addr.c_str(), status.error_message().c_str());
    }
}

}

}

#endif
