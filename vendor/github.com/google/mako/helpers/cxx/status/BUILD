package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "status",
    srcs = [
        "canonical_errors.cc",
        "status.cc",
    ],
    hdrs = [
        "canonical_errors.h",
        "status.h",
    ],
    deps = [
        "@com_github_googleapis_google_cloud_cpp//google/cloud:google_cloud_cpp_common",
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/strings",
    ],
)

cc_library(
    name = "statusor",
    hdrs = ["statusor.h"],
    deps = [
        "@com_github_googleapis_google_cloud_cpp//google/cloud:google_cloud_cpp_common",
    ],
)

cc_library(
    name = "status_matchers",
    testonly = 1,
    srcs = ["status_matchers.cc"],
    hdrs = ["status_matchers.h"],
    deps = [
        ":status",
        ":statusor",
        "@com_google_absl//absl/strings",
        "@com_google_googletest//:gtest",
    ],
)

cc_test(
    name = "status_matchers_test",
    srcs = ["status_matchers_test.cc"],
    deps = [
        ":status",
        ":status_matchers",
        ":statusor",
        "@com_google_googletest//:gtest_main",
    ],
)
