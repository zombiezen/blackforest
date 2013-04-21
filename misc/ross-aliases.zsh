#!/bin/zsh

alias B=blackforest

Bcd() {
    local projpath
    projpath="$(blackforest path "$1")"
    if [ $? -ne 0 ]; then
        echo "Bcd: no path" 1>&2
        return 1
    fi
    if [[ -d "$projpath" ]]; then
        cd "$projpath"
        return $?
    elif [[ -e "$projpath" ]]; then
        cd "${projpath:h}"
        return $?
    else
        echo "Bcd: \"$projpath\" does not exist" 1>&2
        return 1
    fi
}
compdef '_values "blackforest projects" $(blackforest list 2>/dev/null)' Bcd

Binfo() {
    local word="$1"
    shift
    blackforest show "$@" | sed -n 's/^'"$word"':\s*//p'
}

BPUSH() {
    hg push -R "$BLACKFOREST_PATH" "$@"
}

BPULL() {
    hg pull -R "$BLACKFOREST_PATH" -u "$@"
}
