# Analyzer specific proto definitions.

load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

proto_library(
    name = "threshold_analyzer_proto",
    srcs = ["threshold_analyzer.proto"],
    deps = ["//spec/proto:mako_proto"],
)

cc_proto_library(
    name = "threshold_analyzer_cc_proto",
    deps = [":threshold_analyzer_proto"],
)

go_proto_library(
    name = "threshold_analyzer_go_proto",
    importpath = "github.com/google/mako/clients/proto/analyzers/threshold_analyzer_go_proto",
    proto = ":threshold_analyzer_proto",
    deps = ["//spec/proto:mako_go_proto"],
)

proto_library(
    name = "utest_analyzer_proto",
    srcs = ["utest_analyzer.proto"],
    deps = ["//spec/proto:mako_proto"],
)

cc_proto_library(
    name = "utest_analyzer_cc_proto",
    deps = [":utest_analyzer_proto"],
)

go_proto_library(
    name = "utest_analyzer_go_proto",
    importpath = "github.com/google/mako/clients/proto/analyzers/utest_analyzer_go_proto",
    proto = ":utest_analyzer_proto",
    deps = ["//spec/proto:mako_go_proto"],
)

proto_library(
    name = "window_deviation_proto",
    srcs = ["window_deviation.proto"],
    deps = ["//spec/proto:mako_proto"],
)

cc_proto_library(
    name = "window_deviation_cc_proto",
    deps = [":window_deviation_proto"],
)

go_proto_library(
    name = "window_deviation_go_proto",
    importpath = "github.com/google/mako/clients/proto/analyzers/window_deviation_go_proto",
    proto = ":window_deviation_proto",
    deps = ["//spec/proto:mako_go_proto"],
)
