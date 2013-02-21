#!/bin/bash

abspath () { case "$1" in /*)printf "%s\n" "$1";; *)printf "%s\n" "$PWD/$1";; esac; }

root="$(abspath $1)"
shift
created="$(hg -R "$root" log -r0 --template='{date|rfc3339date}\n')"
vcsurl="$(hg -R "$root" paths default | sed 's|^bb://|https://bitbucket.org/|')"

glados create -created="$created" -path="$root" -vcs=hg -vcsurl="$vcsurl" "$@"
exit $?
