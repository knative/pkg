# Quickstore library for C++.
# See quickstore.h and go/mako-quickstore for documentation.
# See quickstore_example_test.cc for example usage.
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "quickstore",
    srcs = [
        "quickstore.cc",
    ],
    hdrs = [
        "quickstore.h",
    ],
    deps = [
        "//proto/quickstore:quickstore_cc_proto",
        "//quickstore/cxx/internal:store",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_glog//:glog",
    ],
)

cc_library(
    name = "mock_quickstore",
    hdrs = ["mock_quickstore.h"],
    deps = [
        ":quickstore",
        "//proto/quickstore:quickstore_cc_proto",
        "@com_google_googletest//:gtest",
    ],
)

cc_test(
    name = "quickstore_example_test",
    size = "enormous",
    srcs = ["quickstore_example_test.cc"],
    tags = [
        "examples",
        "external",
        "manual",  # so the test isn't included in wildcard queries
        "notap",  # Talks to production Mako servers.
    ],
    deps = [
        ":quickstore",
        "//clients/cxx/storage:base_storage",
        "//clients/cxx/storage:google3_storage",
        "//clients/cxx/storage:mako_client",
        "//clients/proto/analyzers:threshold_analyzer_cc_proto",
        "//clients/proto/analyzers:utest_analyzer_cc_proto",
        "//clients/proto/analyzers:window_deviation_cc_proto",
        "//proto/quickstore:quickstore_cc_proto",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/flags:flag",
        "@com_google_absl//absl/time",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)
