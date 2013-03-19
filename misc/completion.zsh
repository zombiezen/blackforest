#!/bin/zsh
# Put this in your ~/.zshrc.
#
# Heavily inspired by go's zsh completion

__glados_complete() {
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
    )
    globalflags+=(
        '-catalog=[path to catalog directory]:file:_path_files -/'
        '-editor=[text editor]'
        '-host=[key for the host]'
        ':command:'
    )
    if (( CURRENT == 2 )); then
        # explain glados commands
        _values 'glados commands' ${commands[@]}
        return
    fi
    __glados_list() {
        local expl projects
        declare -a projects
        projects=($(glados list 2>/dev/null))
        _wanted projects expl 'projects' compadd "$@" - "${projects[@]}"
    }
    __glados_vcs() {
        _values 'glados VCS' 'cvs' 'svn' 'git' 'hg' 'bzr' 'darcs'
        return
    }
    case ${words[2]} in
    init|list|ls|search)
        _arguments : ${globalflags[@]}
        ;;
    show|info)
        _arguments : \
            ${globalflags[@]} \
            '-json[print project as JSON]' \
            '-rfc3339[print dates as RFC3339]' \
            '*:projects:__glados_list'
        ;;
    path|describe|desc)
        _arguments : ${globalflags[@]} ':projects:__glados_list'
        ;;
    delete|del|rm)
        _arguments : ${globalflags[@]} '*:projects:__glados_list'
        ;;
    create)
        _arguments : ${globalflags[@]} \
            '-created=[project creation date, formatted as RFC3339]' \
            '-path=[path of working copy]:file:_path_files -/' \
            '-shortname=[identifier for project]' \
            '-tags=[comma-separated tags to assign to the new project]' \
            '-url=[project homepage]' \
            '-vcs=[type of VCS for project]:vcs:__glados_vcs' \
            '-vcsurl=[project VCS URL]'
        ;;
    update|up)
        _arguments : ${globalflags[@]} \
            '-addtags=[add tags to the project, separated by commas]' \
            '-created=[project creation date, formatted as RFC3339]' \
            '-deltags=[delete tags from the project, separated by commas]' \
            '-name=[human-readable name of project]' \
            '-path=[path of working copy]:file:_path_files -/' \
            "-tags=[set the project's tags, separated by commas]" \
            '-url=[project homepage]' \
            '-vcs=[type of VCS for project]:vcs:__glados_vcs' \
            '-vcsurl=[project VCS URL]' \
            ':projects:__glados_list'
        ;;
    rename|mv)
        _arguments : ${globalflags[@]} ':projects:__glados_list'
        ;;
    describe|desc)
        _arguments : ${globalflags[@]} ':projects:__glados_list'
        ;;
    import)
        _arguments : ${globalflags[@]} '*:file:_files'
        ;;
    checkout|co)
        _arguments : ${globalflags[@]} \
            ':project:__glados_list' \
            ':file:_path_files -/'
        ;;
    esac
}

compdef __glados_complete glados
