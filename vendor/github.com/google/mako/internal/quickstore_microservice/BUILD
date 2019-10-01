load("@io_bazel_rules_docker//cc:image.bzl", "cc_image")

package(default_visibility = ["//:internal"])

licenses(["notice"])

cc_library(
    name = "quickstore_service",
    srcs = ["quickstore_service.cc"],
    hdrs = ["quickstore_service.h"],
    deps = [
        "//clients/cxx/storage:mako_client",
        "//helpers/cxx/status:statusor",
        "//internal/cxx:queue_ifc",
        "//internal/proto:mako_internal_cc_proto",
        "//internal/quickstore_microservice/proto:quickstore_cc_grpc_proto",
        "//internal/quickstore_microservice/proto:quickstore_cc_proto",
        "//quickstore/cxx/internal:store",
        "//spec/cxx:storage",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "quickstore_service_test",
    srcs = ["quickstore_service_test.cc"],
    linkstatic = 1,  # TODO(b/138956597) Fix linker error requiring this.
    deps = [
        ":quickstore_service",
        "//clients/cxx/storage:fake_google3_storage",
        "//internal/cxx:queue",
        "//internal/grpc:grpc++_gce",
        "//internal/quickstore_microservice/proto:quickstore_cc_proto",
        "//proto/quickstore:quickstore_cc_proto",
        "//spec/proto:mako_cc_proto",
        "//testing/cxx:protocol-buffer-matchers",
        "@com_google_absl//absl/memory",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_binary(
    name = "quickstore_microservice_mako",
    srcs = ["main.cc"],
    deps = [
        ":quickstore_service",
        "//internal/cxx:queue",
        "//internal/grpc:grpc++_gce",
        "@com_google_absl//absl/flags:flag",
        "@com_google_absl//absl/flags:parse",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
    ],
)

# See https://github.com/google/mako/blob/master/docs/BUILDING.md#microservice-docker-image
#
# Note, invoking this rule with `bazel run` will load the image into Docker and attempt to run it.
# This will fail due to the lack of credentials (see
# https://github.com/google/mako/blob/master/docs/AUTHENTICATION.md#authenticating-from-a-docker-container).
# There is no way to pass Docker flags in via a `bazel run` invocation.
#
# We would like to set the `legacy_run_behavior` attribute to True so that a `bazel run` invocation
# only loads the image and does not attempt to run it. That is blocked by
# https://github.com/bazelbuild/rules_docker/issues/1085.
#
# Therefore, to load this Docker image, you must instead use `bazel build` to build a tar file that
# can be loaded into Docker. e.g.
#
# $ bazel build //internal/quickstore_microservice:quickstore_microservice_mako_image.tar
# $ docker load -i bazel-bin/internal/quickstore_microservice/quickstore_microservice_mako_image.tar
# $ docker run <docker arguments> bazel/internal/quickstore_microservice:quickstore_microservice_mako_image
#
# See an example of a full command-line at
# https://github.com/google/mako/blob/master/docs/GUIDE.md#quickstore-microservice-as-a-docker-image.
cc_image(
    name = "quickstore_microservice_mako_image",
    args = ["--addr=0.0.0.0:9813"],
    binary = ":quickstore_microservice_mako",
)
