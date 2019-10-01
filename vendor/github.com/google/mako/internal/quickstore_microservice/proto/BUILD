load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

licenses(["notice"])

package(default_visibility = ["//:internal"])

proto_library(
    name = "quickstore_proto",
    srcs = ["quickstore.proto"],
    deps = [
        "//proto/quickstore:quickstore_proto",
        "//spec/proto:mako_proto",
    ],
)

go_proto_library(
    name = "quickstore_go_proto",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "github.com/google/mako/internal/quickstore_microservice/proto/quickstore_go_proto",
    proto = ":quickstore_proto",
    deps = [
        "//proto/quickstore:quickstore_go_proto",
        "//spec/proto:mako_go_proto",
    ],
)

go_proto_library(
    name = "quickstore_go_grpc_pb",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "github.com/google/mako/internal/quickstore_microservice/proto/quickstore_go_grpc_pb",
    proto = ":quickstore_proto",
    deps = [
        "//proto/quickstore:quickstore_go_proto",
        "//spec/proto:mako_go_proto",
    ],
)

load("@com_github_grpc_grpc//bazel:cc_grpc_library.bzl", "cc_grpc_library")

cc_proto_library(
    name = "quickstore_cc_proto",
    deps = [":quickstore_proto"],
)

cc_grpc_library(
    name = "quickstore_cc_grpc_proto",
    srcs = [":quickstore_proto"],
    grpc_only = True,
    deps = [":quickstore_cc_proto"],
)
