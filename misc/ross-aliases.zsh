#!/bin/zsh

alias G=glados

Gcd() {
    local projpath="$(glados path "$1")"
    if [ $? -ne 0 ]; then
        return 1
    fi
    cd "$projpath"
}
compdef '_values "glados projects" $(glados list 2>/dev/null)' Gcd

Ginfo() {
    local word="$1"
    shift
    glados show "$@" | sed -n 's/^'"$word"':\s*//p'
}

alias GPUSH='hg push -R "$GLADOS_PATH"'
alias GPULL='hg pull -R "$GLADOS_PATH" -u'
