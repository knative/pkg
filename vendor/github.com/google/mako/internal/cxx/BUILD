# C++ Internal helpers for mako.
package(default_visibility = ["//:internal"])

licenses(["notice"])

cc_library(
    name = "filter_utils",
    srcs = ["filter_utils.cc"],
    hdrs = ["filter_utils.h"],
    visibility = [
        "//:internal",
        "//contrib/ztest:__pkg__",
    ],
    deps = [
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "filter_utils_test",
    size = "small",
    srcs = ["filter_utils_test.cc"],
    deps = [
        ":filter_utils",
        "//spec/proto:mako_cc_proto",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_library(
    name = "pgmath",
    srcs = ["pgmath.cc"],
    hdrs = ["pgmath.h"],
    visibility = [
        "//:internal",
        "//contrib/ztest:__pkg__",
    ],
)

cc_test(
    name = "pgmath_test",
    size = "small",
    srcs = ["pgmath_test.cc"],
    deps = [
        ":pgmath",
        "@com_google_absl//absl/strings",
        "@com_google_benchmark//:benchmark",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "analyzer_common",
    srcs = ["analyzer_common.cc"],
    hdrs = ["analyzer_common.h"],
    deps = [
        ":filter_utils",
        "//helpers/cxx/status",
        "//helpers/cxx/status:statusor",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
    ],
)

cc_library(
    name = "proto_validation",
    srcs = ["proto_validation.cc"],
    hdrs = ["proto_validation.h"],
    deps = [
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
    ],
)

cc_test(
    name = "proto_validation_test",
    size = "small",
    srcs = ["proto_validation_test.cc"],
    deps = [
        ":proto_validation",
        "//spec/proto:mako_cc_proto",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "backoff",
    srcs = ["backoff.cc"],
    hdrs = ["backoff.h"],
    deps = [
        "@com_google_absl//absl/time",
    ],
)

cc_test(
    name = "backoff_test",
    size = "small",
    srcs = ["backoff_test.cc"],
    deps = [
        ":backoff",
        "@com_google_absl//absl/time",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "clock",
    hdrs = [
        "clock.h",
    ],
    deps = [
        ":clock_external",
    ],
)

cc_library(
    name = "clock_external",
    srcs = [
        "clock_external.cc",
    ],
    hdrs = [
        "clock_external.h",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "@com_google_absl//absl/time",
    ],
)

cc_library(
    name = "clock_mock",
    testonly = True,
    hdrs = ["clock_mock.h"],
    deps = [
        ":clock",
        "@com_google_absl//absl/synchronization",
        "@com_google_absl//absl/time",
        "@com_google_googletest//:gtest",
    ],
)

cc_library(
    name = "analyzer_optimizer",
    srcs = ["analyzer_optimizer.cc"],
    hdrs = ["analyzer_optimizer.h"],
    deps = [
        ":proto_cache",
        "//spec/cxx:analyzer",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
    ],
)

cc_test(
    name = "analyzer_optimizer_test",
    size = "small",
    srcs = ["analyzer_optimizer_test.cc"],
    deps = [
        ":analyzer_optimizer",
        "//clients/cxx/storage:fake_google3_storage",
        "//spec/cxx:analyzer",
        "//spec/cxx:mock_analyzer",
        "//spec/cxx:mock_storage",
        "//spec/cxx:storage",
        "//spec/proto:mako_cc_proto",
        "//testing/cxx:protocol-buffer-matchers",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "queue_ifc",
    hdrs = ["queue_ifc.h"],
    deps = [
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/time",
        "@com_google_absl//absl/types:optional",
        "@com_google_glog//:glog",
    ],
)

cc_library(
    name = "queue",
    hdrs = ["queue.h"],
    deps = [
        ":queue_ifc",
        "@com_google_absl//absl/synchronization",
        "@com_google_absl//absl/time",
        "@com_google_absl//absl/types:optional",
        "@com_google_glog//:glog",
    ],
)

cc_test(
    name = "queue_test",
    size = "small",
    srcs = [
        "queue_test.cc",
    ],
    deps = [
        ":queue",
        "//internal/cxx/load/common:thread_pool_factory",
        "//internal/proto:mako_internal_cc_proto",
        "//testing/cxx:protocol-buffer-matchers",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_absl//absl/time",
        "@com_google_benchmark//:benchmark",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "proto_cache",
    hdrs = ["proto_cache.h"],
    deps = [
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/container:flat_hash_map",
        "@com_google_absl//absl/container:flat_hash_set",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "proto_cache_test",
    size = "small",
    srcs = ["proto_cache_test.cc"],
    deps = [
        ":proto_cache",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_googletest//:gtest_main",
    ],
)
