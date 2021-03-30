workspace(name = "com_github_mingkaic_accretion")

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# === load tenncor dependencies ===

git_repository(
    name = "com_github_mingkaic_tenncor",
    commit = "fe0bd48916e1b1a80be3d24fe032cd62c2b35c45",
    remote = "https://gitlab.com/mingkaic/tenncor.git",
)
local_repository(
    name = "com_github_mingkaic_tenncor",
    path = "/home/mingkaichen/Developer/tenncor",
)

load("@com_github_mingkaic_tenncor//third_party:all.bzl", tenncor_deps = "dependencies")

tenncor_deps()

# === load cppkg dependencies ===

# override cppkg grpc version

http_archive(
    name = "rules_proto_grpc",
    sha256 = "f9b50e672870fe5d60b8b2f3cd1731ceb89a9edef3513d81ba7f7c0d2991b51f",
    strip_prefix = "rules_proto_grpc-3.0.0",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/archive/3.0.0.tar.gz"],
)

load("@com_github_mingkaic_cppkg//third_party:all.bzl", cppkg_deps = "dependencies")

cppkg_deps()

# === boost dependencies ===

load("@com_github_nelhage_rules_boost//:boost/boost.bzl", "boost_deps")

boost_deps()

# === load grpc depedencies ===

# common dependencies
load("@rules_proto_grpc//:repositories.bzl", "rules_proto_grpc_repos", "rules_proto_grpc_toolchains")

rules_proto_grpc_toolchains()

rules_proto_grpc_repos()

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")

grpc_deps()

GOOGLEAPI_BUILD = """
proto_library(
    name = "annotations_proto",
    srcs = glob([
        "google/api/annotations.proto",
        "google/api/http.proto",
    ]),
    visibility = ["//visibility:public"],
    deps = [
        "@com_google_protobuf//:descriptor_proto"
    ],
)"""

http_archive(
    name = "com_google_googleapis",
    url = "https://github.com/googleapis/googleapis/archive/common-protos-1_3_1.zip",
    strip_prefix = "googleapis-common-protos-1_3_1/",
    build_file_content = GOOGLEAPI_BUILD,
)

# c++ dependencies
load("@rules_proto_grpc//cpp:repositories.bzl", rules_proto_grpc_cpp_repos = "cpp_repos")

rules_proto_grpc_cpp_repos()

# python dependencies
load("@rules_proto_grpc//python:repositories.bzl", rules_proto_grpc_python_repos = "python_repos")

rules_proto_grpc_python_repos()

# === load pybind dependencies ===

load("@com_github_pybind_bazel//:python_configure.bzl", "python_configure")

python_configure(name = "local_config_python")

# === development ===

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_grail_bazel_compdb",
    remote = "https://github.com/grailbio/bazel-compilation-database",
    tag = "0.4.5",
)
