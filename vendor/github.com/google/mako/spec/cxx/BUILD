# C++ libaries for accessing mako
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "analyzer",
    srcs = ["analyzer.cc"],
    hdrs = ["analyzer.h"],
    deps = [
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/flags:flag",
        "@com_google_absl//absl/strings",
        "@com_google_farmhash//:farmhash",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_library(
    name = "mock_analyzer",
    testonly = 1,
    hdrs = ["mock_analyzer.h"],
    deps = [
        ":analyzer",
        "@com_google_googletest//:gtest",
    ],
)

cc_library(
    name = "fileio",
    hdrs = ["fileio.h"],
    deps = [
        "@com_google_protobuf//:protobuf",
    ],
)

cc_library(
    name = "aggregator",
    hdrs = ["aggregator.h"],
    deps = [
        ":fileio",
        "//spec/proto:mako_cc_proto",
    ],
)

cc_library(
    name = "downsampler",
    hdrs = ["downsampler.h"],
    deps = [
        ":fileio",
        "//spec/proto:mako_cc_proto",
    ],
)

cc_library(
    name = "dashboard",
    srcs = ["dashboard.cc"],
    hdrs = ["dashboard.h"],
    deps = [
        "//spec/proto:mako_cc_proto",
    ],
)

cc_library(
    name = "storage",
    hdrs = ["storage.h"],
    deps = [
        "//spec/proto:mako_cc_proto",
    ],
)

cc_library(
    name = "mock_storage",
    testonly = 1,
    hdrs = ["mock_storage.h"],
    deps = [
        ":storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_googletest//:gtest",
    ],
)
