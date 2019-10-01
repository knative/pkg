load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])  # Apache 2.0

proto_library(
    name = "mako_proto",
    srcs = ["mako.proto"],
)

cc_proto_library(
    name = "mako_cc_proto",
    deps = [":mako_proto"],
)

go_proto_library(
    name = "mako_go_proto",
    importpath = "github.com/google/mako/spec/proto/mako_go_proto",
    proto = ":mako_proto",
)
