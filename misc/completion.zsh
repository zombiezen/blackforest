#!/bin/zsh
# Put this in your ~/.zshrc.
#
# Heavily inspired by go's zsh completion

__blackforest_complete() {
    typeset -a commands globalflags
    commands+=(
        'init[create a catalog]'
        'list[list project short names]'
        'ls[list project short names]'
        "path[print a project's local path]"
        'show[print projects]'
        'info[print projects]'
        'create[create a project]'
        'update[change project fields]'
        'up[change project fields]'
        'describe[edit project description]'
        'desc[edit project description]'
        "rename[change a project's short name]"
        "mv[change a project's short name]"
        'delete[delete projects]'
        'del[delete projects]'
        'rm[delete projects]'
        'import[import project(s) from JSON]'
        'checkout[check out project from version control]'
        'co[check out project from version control]'
        'search[full text search for projects]'
        'web[run web server]'
        'verify[check a catalog for consistency]'
    )
    globalflags+=(
        '-catalog=[path to catalog directory]:file:_path_files -/'
        '-editor=[text editor]'
        '-host=[key for the host]'
        ':command:'
    )
    if (( CURRENT == 2 )); then
        # explain blackforest commands
        _values 'blackforest commands' ${commands[@]}
        return
    fi
    __blackforest_list() {
        local expl projects
        declare -a projects
        projects=($(blackforest list 2>/dev/null))
        _wanted projects expl 'projects' compadd "$@" - "${projects[@]}"
    }
    __blackforest_vcs() {
        _values 'blackforest VCS' 'cvs' 'svn' 'git' 'hg' 'bzr' 'darcs'
        return
    }
    case ${words[2]} in
    init|list|ls|verify|search)
        _arguments : ${globalflags[@]}
        ;;
    show|info)
        _arguments : \
            ${globalflags[@]} \
            '-json[print project as JSON]' \
            '-rfc3339[print dates as RFC3339]' \
            '*:projects:__blackforest_list'
        ;;
    path|describe|desc)
        _arguments : ${globalflags[@]} ':projects:__blackforest_list'
        ;;
    delete|del|rm)
        _arguments : ${globalflags[@]} '*:projects:__blackforest_list'
        ;;
    create)
        _arguments : ${globalflags[@]} \
            '-created=[project creation date, formatted as RFC3339]' \
            '-description=[human-readable project description]' \
            '-path=[path of working copy]:file:_files' \
            '-shortname=[identifier for project]' \
            '-tags=[comma-separated tags to assign to the new project]' \
            '-url=[project homepage]' \
            '-vcs=[type of VCS for project]:vcs:__blackforest_vcs' \
            '-vcsurl=[project VCS URL]'
        ;;
    update|up)
        _arguments : ${globalflags[@]} \
            '-addtags=[add tags to the project, separated by commas]' \
            '-created=[project creation date, formatted as RFC3339]' \
            '-deltags=[delete tags from the project, separated by commas]' \
            '-description=[human-readable project description]' \
            '-name=[human-readable name of project]' \
            '-path=[path of working copy]:file:_files' \
            "-tags=[set the project's tags, separated by commas]" \
            '-url=[project homepage]' \
            '-vcs=[type of VCS for project]:vcs:__blackforest_vcs' \
            '-vcsurl=[project VCS URL]' \
            ':projects:__blackforest_list'
        ;;
    rename|mv)
        _arguments : ${globalflags[@]} ':projects:__blackforest_list'
        ;;
    describe|desc)
        _arguments : ${globalflags[@]} ':projects:__blackforest_list'
        ;;
    import)
        _arguments : ${globalflags[@]} '*:file:_files'
        ;;
    checkout|co)
        _arguments : ${globalflags[@]} \
            ':project:__blackforest_list' \
            ':file:_path_files -/'
        ;;
    web)
        _arguments : ${globalflags[@]} \
            '-listen=[address to listen for HTTP]' \
            '-refresh=[interval between catalog cache refreshes]' \
            '-staticdir=[static directory]:file:_path_files -/' \
            '-templatedir=[template directory]:file:_path_files -/'
        ;;
    esac
}

compdef __blackforest_complete blackforest
