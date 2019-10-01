package(default_visibility = ["//:internal"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_binary", "go_test")

go_binary(
    name = "mako",
    srcs = ["mako.go"],
    deps = [
        ":lib",
        "//clients/go/storage:mako",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_golang_subcommands//:go_default_library",
    ],
)

go_library(
    name = "lib",
    srcs = ["lib.go"],
    importpath = "github.com/google/mako/internal/tools/cli/lib",
    deps = [
        "//spec/go:mako",
        "//spec/proto:mako_go_proto",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_golang_subcommands//:go_default_library",
    ],
)

go_test(
    name = "lib_test",
    size = "small",
    srcs = ["lib_test.go"],
    data = [
        "//examples/benchmark_info:create_benchmark.config",
    ],
    embed = [":lib"],
    importpath = "github.com/google/mako/internal/tools/cli/lib_test",
    deps = [
        "//clients/go/storage:fakestorage",
        "//spec/proto:mako_go_proto",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_golang_subcommands//:go_default_library",
        "@io_bazel_rules_go//go/tools/bazel:go_default_library",
    ],
)
