package(default_visibility = ["//:internal"])

licenses(["notice"])

cc_library(
    name = "cleanup",
    hdrs = ["cleanup.h"],
    deps = [
        "@com_google_absl//absl/base:core_headers",
    ],
)

cc_test(
    name = "cleanup_test",
    size = "small",
    srcs = ["cleanup_test.cc"],
    deps = [
        ":cleanup",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "googleinit",
    hdrs = ["googleinit.h"],
    deps = [
    ],
)

cc_library(
    name = "stringutil",
    srcs = ["stringutil.cc"],
    hdrs = ["stringutil.h"],
    deps = [
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
    ],
)

cc_test(
    name = "stringutil_test",
    size = "small",
    srcs = ["stringutil_test.cc"],
    deps = [
        ":stringutil",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)
