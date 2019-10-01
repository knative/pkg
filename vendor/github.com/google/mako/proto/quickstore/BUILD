# See language specific directory and go/mako-quickstore for documentation.
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "quickstore_proto",
    srcs = ["quickstore.proto"],
    deps = [
        "//clients/proto/analyzers:threshold_analyzer_proto",
        "//clients/proto/analyzers:utest_analyzer_proto",
        "//clients/proto/analyzers:window_deviation_proto",
        "//spec/proto:mako_proto",
    ],
)

cc_proto_library(
    name = "quickstore_cc_proto",
    deps = [":quickstore_proto"],
)

go_proto_library(
    name = "quickstore_go_proto",
    importpath = "github.com/google/mako/proto/quickstore/quickstore_go_proto",
    proto = ":quickstore_proto",
    deps = [
        "//clients/proto/analyzers:threshold_analyzer_go_proto",
        "//clients/proto/analyzers:utest_analyzer_go_proto",
        "//clients/proto/analyzers:window_deviation_go_proto",
        "//spec/proto:mako_go_proto",
    ],
)
