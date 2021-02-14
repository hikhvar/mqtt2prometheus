#!/bin/sh

go-fuzz-build -o fuzz-target && go-fuzz -bin fuzz-target

rm -rf fuzz-target
