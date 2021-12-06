#!/bin/bash -x

case "$OSTYPE" in
  darwin*)  SOCATTTY=true ;; 
  linux*)   SOCATTTY=true ;;
  *)        SOCATTTY=false ;;
esac

if [[ "$SOCATTTY" == "true" ]]; then
    socat -d -d \
    pty,raw,nonblock,echo=0,iexten=0,link=/tmp/tty.fake.slave \
    pty,raw,nonblock,echo=0,iexten=0,link=/tmp/tty.fake.master &

    SOCATPID=$!

    trap "kill -9 $SOCATPID" EXIT
fi

MODULE=github.com/samuelventura/go-serial

go clean -testcache
go test
