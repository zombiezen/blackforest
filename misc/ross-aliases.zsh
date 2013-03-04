#!/bin/zsh

alias G=glados

gcd() { cd "$(glados path "$1")"; return $? }
compdef '_values "glados projects" $(glados list 2>/dev/null)' gcd

ginfo() { glados show "$1" | sed -n 's/^'"$2"':\s*//p' }

alias GPUSH='hg push -R "$GLADOS_PATH"'
alias GPULL='hg pull -R "$GLADOS_PATH" -u'
