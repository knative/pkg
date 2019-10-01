package(default_visibility = ["//:internal"])

licenses(["notice"])

proto_library(
    name = "mako_internal_proto",
    srcs = ["mako_internal.proto"],
    deps = ["//spec/proto:mako_proto"],
)

cc_proto_library(
    name = "mako_internal_cc_proto",
    deps = [":mako_internal_proto"],
)
