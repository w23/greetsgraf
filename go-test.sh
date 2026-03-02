#!/bin/sh

CGO_CFLAGS="-D_LARGEFILE64_SOURCE" CGO_ENABLED=1 go test -tags "sqlite_omit_load_extension sqlite_fts5" $@
