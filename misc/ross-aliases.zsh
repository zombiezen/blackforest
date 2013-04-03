#!/bin/zsh

alias G=glados

Gcd() {
    local projpath
    projpath="$(glados path "$1")"
    if [ $? -ne 0 ]; then
        echo "Gcd: no path" 1>&2
        return 1
    fi
    if [[ -d "$projpath" ]]; then
        cd "$projpath"
        return $?
    elif [[ -e "$projpath" ]]; then
        cd "${projpath:h}"
        return $?
    else
        echo "Gcd: \"$projpath\" does not exist" 1>&2
        return 1
    fi
}
compdef '_values "glados projects" $(glados list 2>/dev/null)' Gcd

Ginfo() {
    local word="$1"
    shift
    glados show "$@" | sed -n 's/^'"$word"':\s*//p'
}

GPUSH() {
    hg push -R "$GLADOS_PATH" "$@"
}

GPULL() {
    hg pull -R "$GLADOS_PATH" -u "$@"
}
