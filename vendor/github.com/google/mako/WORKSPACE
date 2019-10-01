load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository", "new_git_repository")


# proto_library, cc_proto_library, and java_proto_library rules implicitly
# depend on @com_google_protobuf for protoc and proto runtimes.
# This statement defines the @com_google_protobuf repo.
http_archive(
    name = "com_google_protobuf",
    #sha256 = "f976a4cd3f1699b6d20c1e944ca1de6754777918320c719742e1674fcf247b7e",
    strip_prefix = "protobuf-3.8.0",
    urls = ["https://github.com/google/protobuf/archive/v3.8.0.zip"],
    sha256 = "1e622ce4b84b88b6d2cdf1db38d1a634fe2392d74f0b7b74ff98f3a51838ee53",
)

http_archive(
    name = "bazel_skylib",
    strip_prefix = "bazel-skylib-0.9.0",
    sha256 = "9245b0549e88e356cd6a25bf79f97aa19332083890b7ac6481a2affb6ada9752",
    urls = ["https://github.com/bazelbuild/bazel-skylib/archive/0.9.0.tar.gz"],
)

# Abseil
http_archive(
    name = "com_google_absl",
    strip_prefix = "abseil-cpp-3c98fcc0461bd2a4b9c149d4748a7373a225cf4b",
    url = "https://github.com/abseil/abseil-cpp/archive/3c98fcc0461bd2a4b9c149d4748a7373a225cf4b.zip",
    sha256 = "98b6dcedd976a95567625f30685cb049f5e9c3ed9e6d4dce97dcf8f24517add7",
)

# GoogleTest/GoogleMock framework. Used by most unit-tests.
# https://github.com/google/googletest/commits/2134e3fd857d952e03ce76064fad5ac6e9036104
# Jul 25, 2019
http_archive(
    name = "com_google_googletest",
    urls = ["https://github.com/google/googletest/archive/2134e3fd857d952e03ce76064fad5ac6e9036104.zip"],
    strip_prefix = "googletest-2134e3fd857d952e03ce76064fad5ac6e9036104",
    sha256 = "c36c2757d2aaf6beba78bf899a35ca6eab2e1fc28834d435325b673311a114f7",
)

# glog
# TODO(b/134946989) Switch to using Abseil logging
git_repository(
    name = "com_google_glog",
    commit = "e364e754a60af6f0eadd9902c4e76ecc060fee9c",
    remote = "https://github.com/google/glog.git",
    shallow_since = "1540324206 +0200",
)

# gflags
# Used by glog.
# TODO(b/134946989) Can go away when glog goes away.
http_archive(
    name = "com_github_gflags_gflags",
    sha256 = "6e16c8bc91b1310a44f3965e616383dbda48f83e8c1eaa2370a215057b00cabe",
    strip_prefix = "gflags-77592648e3f3be87d6c7123eb81cbad75f9aef5a",
    urls = [
        "https://mirror.bazel.build/github.com/gflags/gflags/archive/77592648e3f3be87d6c7123eb81cbad75f9aef5a.tar.gz",
        "https://github.com/gflags/gflags/archive/77592648e3f3be87d6c7123eb81cbad75f9aef5a.tar.gz",
    ],
)

# farmhash
new_git_repository(
    name = "com_google_farmhash",
    build_file_content = """
package(default_visibility = ["//visibility:public"])

cc_library(
    name = "farmhash",
    hdrs = ["src/farmhash.h"],
    srcs = ["src/farmhash.cc"],
    deps = [
       # "@com_google_absl//base:core_headers",
       # "@com_google_absl//base:endian",
       # "@com_google_absl//numeric:int128",
    ],
)""",
    commit = "2f0e005b81e296fa6963e395626137cf729b710c",
    remote = "https://github.com/google/farmhash.git",
    shallow_since = "1509400690 -0700",
)

# Google Benchmark
git_repository(
    name = "com_google_benchmark",
    remote = "https://github.com/google/benchmark",
    commit = "090faecb454fbd6e6e17a75ef8146acb037118d4", # v1.5.0
    shallow_since = "1557776538 +0300"
)

# Google Cloud CPP
http_archive(
    name = "com_github_googleapis_google_cloud_cpp",
    url = "http://github.com/googleapis/google-cloud-cpp/archive/v0.11.0.tar.gz",
    strip_prefix = "google-cloud-cpp-0.11.0",
    sha256 = "3abe2cf553ce33ff58d23848ae716cd2fcabfd454b89f6f65a92ed261244c1df",
)
load("@com_github_googleapis_google_cloud_cpp//bazel:google_cloud_cpp_deps.bzl", "google_cloud_cpp_deps")
google_cloud_cpp_deps()

# libcurl
http_archive(
    name = "com_github_curl_curl",
    urls = [
        "https://mirror.bazel.build/curl.haxx.se/download/curl-7.49.1.tar.gz",
    ],
    sha256 = "ff3e80c1ca6a068428726cd7dd19037a47cc538ce58ef61c59587191039b2ca6",
    strip_prefix = "curl-7.49.1",
    build_file = "@//:curl.BUILD",
)

http_archive(
    name = "boringssl",
    strip_prefix = "boringssl-18637c5f37b87e57ebde0c40fe19c1560ec88813",
    url = "https://github.com/google/boringssl/archive/18637c5f37b87e57ebde0c40fe19c1560ec88813.zip",
    sha256 = "bd923e59fca0d2b50db09af441d11c844c5e882a54c68943b7fc39a8cb5dd211",
)

# grpc
# Must be after boringssl so grpc doesn't pull in its own version of boringssl
http_archive(
    name = "com_github_grpc_grpc",
    urls = ["https://github.com/grpc/grpc/archive/v1.22.0.tar.gz"],
    strip_prefix = "grpc-1.22.0",
    sha256 = "11ac793c562143d52fd440f6549588712badc79211cdc8c509b183cb69bddad8",
)

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")
grpc_deps()

# ===== re2 =====
http_archive(
    name = "com_googlesource_code_re2",
    sha256 = "5306526bcdf35ff34c67913bef8f7b15a3960f4f0ab3a2b6a260af4f766902d4",
    strip_prefix = "re2-c4f65071cc07eb34d264b25f7b9bbb679c4d5a5a",
    urls = [
        "https://mirror.bazel.build/github.com/google/re2/archive/c4f65071cc07eb34d264b25f7b9bbb679c4d5a5a.tar.gz",
        "https://github.com/google/re2/archive/c4f65071cc07eb34d264b25f7b9bbb679c4d5a5a.tar.gz",
    ],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
protobuf_deps()

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.18.6/rules_go-0.18.6.tar.gz"],
    sha256 = "f04d2373bcaf8aa09bccb08a98a57e721306c8f6043a2a0ee610fd6853dcde3d",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "com_github_golang_protobuf",
    importpath = "github.com/golang/protobuf",
    tag = "v1.3.1",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    tag = "v1.20.1",
)

go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    tag = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
)

go_repository(
    name = "com_github_golang_subcommands",
    importpath = "github.com/google/subcommands",
    tag = "v1.0.1",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "e513c0ac6534810eb7a14bf025a0f159726753f97f74ab7863c650d26e01d677",
    strip_prefix = "rules_docker-0.9.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.9.0.tar.gz"],
)
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
container_repositories()
load(
    "@io_bazel_rules_docker//cc:image.bzl",
    _cc_image_repos = "repositories",
)
_cc_image_repos()

# Swig 3.0.8
http_archive(
    name = "swig",
    build_file = "//:swig.BUILD",
    sha256 = "58a475dbbd4a4d7075e5fe86d4e54c9edde39847cdb96a3053d87cb64a23a453",
    strip_prefix = "swig-3.0.8",
    urls = [
        "https://mirror.bazel.build/ufpr.dl.sourceforge.net/project/swig/swig/swig-3.0.8/swig-3.0.8.tar.gz",
        "http://ufpr.dl.sourceforge.net/project/swig/swig/swig-3.0.8/swig-3.0.8.tar.gz",
        "http://pilotfiber.dl.sourceforge.net/project/swig/swig/swig-3.0.8/swig-3.0.8.tar.gz",
    ],
)

# PCRE, used by SWIG
http_archive(
    name = "pcre",
    build_file = "//:pcre.BUILD",
    sha256 = "69acbc2fbdefb955d42a4c606dfde800c2885711d2979e356c0636efde9ec3b5",
    strip_prefix = "pcre-8.42",
    urls = [
        "https://mirror.bazel.build/ftp.exim.org/pub/pcre/pcre-8.42.tar.gz",
        "http://ftp.exim.org/pub/pcre/pcre-8.42.tar.gz",
    ],
)
