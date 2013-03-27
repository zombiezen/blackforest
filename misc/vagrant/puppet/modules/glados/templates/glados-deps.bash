#!/bin/bash

GO=/usr/local/go/bin/go
format='{{range .Deps}}{{.}}'$'\n''{{end}}'
deps="$($GO list -f "$format" <%=importpath%> | sed -rn '\:^bitbucket\.org/zombiezen/glados($|/):!p')"
$GO get -u $deps
