#!/bin/bash

die() {
    echo "$@" 1>&2
    exit 1
}

if [ "$(whoami)" != root ]
then
    die "must be run as root"
fi

GO=/usr/local/go/bin/go

initctl stop glados || die "** can't stop service"
sudo -u <%=user%> GOPATH='<%=gopath%>' $GO install <%=importpath%> || die "** build failed"
initctl start glados || die "** can't restart service"
