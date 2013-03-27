class glados {
    $importpath = "bitbucket.org/zombiezen/glados"
    $go_url = "http://go.googlecode.com/files/go1.0.3.linux-386.tar.gz"
    $user = "vagrant"
    $gopath = "/home/$user/go"
    $catalog = "/srv/glados-catalog"

    group { $user:
        ensure => present,
    }

    user { $user:
        ensure => present,
        gid => $user,
    }

    File {
        owner => $user,
        group => $user,
    }

    Exec {
        user => $user,
    }

    package { ["git-core", "mercurial", "wget"]:
        ensure => installed,
    }

    exec { "fetch-go":
        require => Package["wget"],
        command => "/usr/bin/wget -q -O - $go_url | /bin/tar xz -C /usr/local",
        creates => "/usr/local/go/bin/go",
        user => "root",
    }

    file {
        ["$gopath",
         "$gopath/src",
         "$gopath/src/bitbucket.org",
         "$gopath/src/bitbucket.org/zombiezen"]:
            ensure => directory;

        "$gopath/src/$importpath":
            ensure => link,
            target => "/vagrant";
    }

    file { "/tmp/glados-deps.bash":
        ensure => file,
        content => template("glados/glados-deps.bash"),
        owner => "root",
        group => "root",
        mode => 755,
    }

    exec { "glados-deps":
        require => [
            Exec["fetch-go"],
            Package["git-core"],
            Package["mercurial"],
            File["/tmp/glados-deps.bash"],
            File["$gopath/src/$importpath"],
        ],
        command => "/tmp/glados-deps.bash",
        environment => "GOPATH=$gopath",
    }

    file { "/home/$user/glados-reload":
        ensure => file,
        content => template("glados/glados-reload.bash"),
        mode => 755,
    }

    exec { "glados-install":
        require => Exec["glados-deps"],
        command => "/usr/local/go/bin/go install $importpath",
        environment => "GOPATH=$gopath",
        creates => "$gopath/bin/glados",
    }

    file { "/etc/init/glados.conf":
        mode => 600,
        owner => "root",
        group => "root",
        content => template("glados/glados.conf"),
    }

    exec { "glados-init":
        require => Exec["glados-install"],
        command => "$gopath/bin/glados init -catalog=\"$catalog\"",
        creates => $catalog,
        user => "root",
    }

    service { "glados":
        require => [File["/etc/init/glados.conf"], Exec["glados-init"]],
        ensure => running,
    }
}
