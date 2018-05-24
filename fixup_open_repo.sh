#! /bin/bash

# This script fixes the open source mish repo. It's meant to be read just by the
# Tilter doing the export.

# This repo is created from our internal windmill repo by copybara
# (https://github.com/google/copybara)
# We export just the code needed for mish.
# As a result, we have to then fixup what's public to include, e.g.:
# *) Vendored code
# *) Generated protobufs (internally we use a build system and don't check them in)

# After copybara export, you'll want to:
# clone the mish repo it generates
# run this script
# test that mish still works
# commit the result
# push

set -ex

dep ensure
find . -name *.pb.go | grep -v vendor | xargs rm -f
protoc --go_out=plugins=grpc:../../../ -I. bridge/fs/proto/*.proto
protoc --go_out=plugins=grpc:../../../ -I. data/db/proto/*.proto
protoc --go_out=plugins=grpc:../../../ -I. data/proto/*.proto
