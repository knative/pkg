# Go interfaces for mako
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "mako",
    srcs = [
        "storage.go",
    ],
    importpath = "github.com/google/mako/spec/go/mako",
    deps = [
        "//spec/proto:mako_go_proto",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
