# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# see the license for the specific language governing permissions and
# limitations under the license.
"""Bazel rules for building swig files."""

load("@io_bazel_rules_go//go:def.bzl", "go_library")

def _get_repository_roots(files):
    result = {}
    for f in files.to_list():
        root = f.root.path
        if root:
            if root not in result:
                result[root] = 0
            result[root] -= 1
        work = f.owner.workspace_root
        if work:
            if root:
                root += "/"
            root += work
        if root:
            if root not in result:
                result[root] = 0
            result[root] -= 1
    return [k for _, k in sorted([(v, k) for k, v in result.items()])]

def _go_wrap_cc_impl(ctx):
    srcs = ctx.files.srcs
    if len(srcs) != 1:
        fail("Exactly one SWIG source file label must be specified.", "srcs")
    module_name = ctx.attr.module_name
    src = ctx.files.srcs[0]
    inputs = [src]
    inputs += ctx.files.swig_includes

    for dep in ctx.attr.deps:
        inputs += dep[CcInfo].compilation_context.headers.to_list()
    inputs += ctx.files._swiglib
    inputs += ctx.files.toolchain_deps
    inputs = depset(inputs)
    swig_include_dirs = depset(_get_repository_roots(inputs) + sorted([f.dirname for f in ctx.files._swiglib]))
    args = [
        "-go",
        "-cgo",
        "-c++",
        "-module",
        module_name,
        "-o",
        ctx.outputs.cc_out.path,
        "-intgosize",
        "64",
        "-outdir",
        ctx.outputs.go_out.dirname,
    ]
    args += ["-l" + f.path for f in ctx.files.swig_includes]
    args += ["-I" + i for i in swig_include_dirs.to_list()]
    args += [src.path]
    outputs = [ctx.outputs.cc_out, ctx.outputs.go_out]
    ctx.actions.run(
        executable = ctx.executable._swig,
        arguments = args,
        inputs = inputs,
        outputs = outputs,
        mnemonic = "GoSwig",
        progress_message = "SWIGing " + src.path,
    )
    return [DefaultInfo(files = depset(outputs))]

_go_wrap_cc = rule(
    attrs = {
        "srcs": attr.label_list(
            mandatory = True,
            allow_files = True,
        ),
        "swig_includes": attr.label_list(
            allow_files = True,
        ),
        "deps": attr.label_list(
            allow_files = True,
        ),
        "toolchain_deps": attr.label_list(
            allow_files = True,
        ),
        "module_name": attr.string(mandatory = True),
        "_swig": attr.label(
            default = Label("@swig//:swig"),
            executable = True,
            cfg = "host",
        ),
        "_swiglib": attr.label(
            default = Label("@swig//:templates"),
            allow_files = True,
        ),
    },
    outputs = {
        "cc_out": "%{module_name}.cc",
        "go_out": "%{module_name}.go",
    },
    implementation = _go_wrap_cc_impl,
)

def mako_go_wrap_cc(
        name,
        srcs,
        importpath,
        swig_includes = [],
        cdeps = [],
        godeps = [],
        copts = [],
        **kwargs):
    """Mako wrapper for building SWIG files for Go with Bazel"""

    cc_library_name = "_" + name
    _go_wrap_cc(
        name = name + "_go_wrap",
        srcs = srcs,
        module_name = name,
        swig_includes = swig_includes,
        toolchain_deps = ["@bazel_tools//tools/cpp:current_cc_toolchain"],
        deps = cdeps,
    )
    native.cc_library(
        name = cc_library_name,
        srcs = [name + ".cc"],
        copts = copts + [
            "-Wno-sign-compare",
            "-Wno-write-strings",
            "-Wno-unused-function",
        ],
        deps = cdeps,
        **kwargs
    )
    go_library(
        name = name,
        cgo = 1,
        srcs = [":" + name + ".go"],
        cdeps = [":" + cc_library_name],
        importpath = importpath,
        deps = godeps,
    )
