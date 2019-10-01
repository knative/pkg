#Implementors of mako dashboard interface.
package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "standard_dashboard",
    srcs = ["standard_dashboard.cc"],
    hdrs = ["standard_dashboard.h"],
    deps = [
        "//spec/cxx:dashboard",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/base",
        "@com_google_absl//absl/strings",
    ],
)

cc_test(
    name = "standard_dashboard_test",
    size = "small",
    srcs = ["standard_dashboard_test.cc"],
    deps = [
        ":standard_dashboard",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/types:optional",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
    ],
)
