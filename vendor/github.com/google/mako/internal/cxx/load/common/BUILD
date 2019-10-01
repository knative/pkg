# C++ Common Load components
package(default_visibility = ["//:internal"])

licenses(["notice"])

cc_library(
    name = "executor",
    hdrs = ["executor.h"],
    visibility = [
        "//:internal",
        "//helpers/cxx/streamz_metric_collector:__pkg__",
    ],
    deps = [
        ":thread_pool_factory",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/synchronization",
        "@com_google_glog//:glog",
    ],
)

cc_test(
    name = "executor_test",
    size = "small",
    srcs = ["executor_test.cc"],
    deps = [
        ":executor",
        "@com_google_absl//absl/synchronization",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "thread_pool",
    hdrs = ["thread_pool.h"],
    visibility = ["//visibility:private"],
    deps = [
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/synchronization",
    ],
)

cc_test(
    name = "thread_pool_test",
    srcs = ["thread_pool_test.cc"],
    deps = [
        ":thread_pool",
        "@com_google_absl//absl/synchronization",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "thread_pool_factory",
    srcs = ["thread_pool_factory.cc"],
    hdrs = ["thread_pool_factory.h"],
    deps = [
        ":thread_pool",
        "@com_google_absl//absl/memory",
    ],
)

cc_library(
    name = "run_analyzers",
    srcs = ["run_analyzers.cc"],
    hdrs = ["run_analyzers.h"],
    deps = [
        "//helpers/cxx/status",
        "//internal/cxx:analyzer_optimizer",
        "//internal/proto:mako_internal_cc_proto",
        "//spec/cxx:aggregator",
        "//spec/cxx:analyzer",
        "//spec/cxx:dashboard",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "run_analyzers_test",
    size = "small",
    srcs = ["run_analyzers_test.cc"],
    deps = [
        ":run_analyzers",
        "//clients/cxx/analyzers:threshold_analyzer",
        "//clients/cxx/dashboard:standard_dashboard",
        "//clients/cxx/storage:fake_google3_storage",
        "//clients/proto/analyzers:threshold_analyzer_cc_proto",
        "//internal/proto:mako_internal_cc_proto",
        "//spec/cxx:analyzer",
        "//spec/cxx:dashboard",
        "//spec/cxx:mock_analyzer",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "//testing/cxx:protocol-buffer-matchers",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)
