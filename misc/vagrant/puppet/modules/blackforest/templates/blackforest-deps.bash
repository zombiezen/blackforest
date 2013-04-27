#!/bin/bash

GO=/usr/local/go/bin/go
format='{{range .Deps}}{{.}}'$'\n''{{end}}'
deps="$($GO list -f "$format" <%=importpath%> | sed -rn '\:^bitbucket\.org/zombiezen/blackforest($|/):!p')"
$GO get -u $deps
