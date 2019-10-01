# Google3 implementation of mako storage interface.
package(
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

cc_library(
    name = "google3_storage",
    srcs = ["google3_storage.cc"],
    hdrs = ["google3_storage.h"],
    deps = [
        "//internal/cxx:proto_validation",
        "//internal/cxx/storage_client:retry_strategy",
        "//internal/cxx/storage_client:retrying_storage_request",
        "//internal/cxx/storage_client:transport",
        "//internal/proto:mako_internal_cc_proto",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/container:flat_hash_map",
        "@com_google_absl//absl/flags:flag",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/time",
        "@com_google_glog//:glog",
    ],
)

cc_library(
    name = "base_storage",
    hdrs = ["base_storage_client.h"],
    visibility = ["//:internal"],
    deps = [
        ":google3_storage",
    ],
)

cc_library(
    name = "fake_google3_storage",
    srcs = ["fake_google3_storage.cc"],
    hdrs = ["fake_google3_storage.h"],
    deps = [
        "//internal/cxx:proto_validation",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/synchronization",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "fake_google3_storage_test",
    size = "small",
    srcs = ["fake_google3_storage_test.cc"],
    deps = [
        ":fake_google3_storage",
        "//testing/cxx:protocol-buffer-matchers",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "mako_client",
    srcs = ["mako_client.cc"],
    hdrs = ["mako_client.h"],
    deps = [
        ":base_storage",
        ":google3_storage",
        "@com_google_absl//absl/flags:flag",
        "@com_google_absl//absl/strings",
        "//helpers/cxx/status:statusor",
        "//internal/cxx/storage_client:google_oauth_fetcher",
        "//internal/cxx/storage_client:http_transport",
    ] + [
    ],
)

cc_library(
    name = "mako_client_internal",
    hdrs = ["mako_client_internal.h"],
    visibility = [
        "//clients/cxx/storage/go:__pkg__",
        "//clients/cxx/storage/python:__pkg__",
    ],
    deps = [
        ":mako_client",
        "//clients/cxx/storage:google3_storage",
        "//helpers/cxx/status",
        "//helpers/cxx/status:statusor",
        "@com_google_absl//absl/strings",
    ],
)

cc_test(
    name = "mako_client_test",
    srcs = ["mako_client_test.cc"],
    data = [
    ],
    deps = [
        ":google3_storage",
        ":mako_client",
        "//internal/cxx/storage_client:google_oauth_fetcher",
        "//internal/cxx/storage_client:http_transport",
        "//internal/cxx/storage_client:oauth_token_provider",
        "//internal/cxx/storage_client:transport",
        "@com_google_absl//absl/flags:flag",
        "@com_google_googletest//:gtest_main",
    ],
)
