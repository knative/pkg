# Go libraries for mako
package(default_visibility = ["//:internal"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "wrappedstorage",
    srcs = ["wrappedstorage.go"],
    importpath = "github.com/google/mako/internal/go/wrappedstorage",
    deps = [
        "//spec/proto:mako_go_proto",
    ],
)
