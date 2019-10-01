# Go storage mako library
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "g3storage",
    srcs = ["google3_storage.go"],
    importpath = "github.com/google/mako/clients/go/storage/g3storage",
    deps = [
        "//clients/cxx/storage/go:g3storage_wrap",
        "//internal/go:wrappedstorage",
    ],
)

go_library(
    name = "mako",
    srcs = ["mako.go"],
    importpath = "github.com/google/mako/clients/go/storage/mako",
    deps = [
        ":g3storage",
        "//clients/cxx/storage/go:mako_client_wrap",
    ],
)

go_library(
    name = "fakestorage",
    srcs = ["fake_storage.go"],
    importpath = "github.com/google/mako/clients/go/storage/fakestorage",
    deps = [
        "//clients/cxx/storage/go:fakestorage_wrap",
        "//internal/go:wrappedstorage",
        "//spec/proto:mako_go_proto",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "fakestorage_test",
    srcs = ["fake_storage_test.go"],
    embed = [":fakestorage"],
    importpath = "github.com/google/mako/clients/go/storage/fakestorage_test",
    deps = [
        "//spec/go:mako",
        "//spec/proto:mako_go_proto",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
