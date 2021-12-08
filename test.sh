#!/bin/bash -x

case "$OSTYPE" in
  darwin*)  SOCATTTY=true ;; 
  linux*)   SOCATTTY=true ;;
  *)        SOCATTTY=false ;;
esac

if [[ "$SOCATTTY" == "true" ]]; then
    socat -d -d \
    pty,link=/tmp/tty.slave \
    pty,link=/tmp/tty.master &

    SOCATPID=$!

    trap "kill -9 $SOCATPID" EXIT
fi

MODULE=github.com/samuelventura/go-serial

go clean -testcache
go test
