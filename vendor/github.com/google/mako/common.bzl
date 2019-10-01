""" A place to put things that are reused throughout Mako BUILD files. """

MAKO_INTERNAL_PROD_DEFINE = select({
    "//tools/cc_target_os:gce": [],
    "//conditions:default": ["MAKO_INTERNAL"],
})
