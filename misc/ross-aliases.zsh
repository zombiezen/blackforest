#!/bin/zsh

alias G=glados

gcd() { cd "$(glados path "$1")"; return $? }
compdef '_values "glados projects" $(glados list 2>/dev/null)' gcd

ginfo { ginfo() { glados show "$1" | sed -n 's/^'"$2"':\s*//p'; exit $? }
