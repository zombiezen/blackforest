class blackforest {
    $importpath = "bitbucket.org/zombiezen/blackforest"
    $go_url = "http://go.googlecode.com/files/go1.0.3.linux-386.tar.gz"
    $user = "vagrant"
    $gopath = "/home/$user/go"
    $catalog = "/srv/blackforest-catalog"

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

    file { "/tmp/blackforest-deps.bash":
        ensure => file,
        content => template("blackforest/blackforest-deps.bash"),
        owner => "root",
        group => "root",
        mode => 755,
    }

    exec { "blackforest-deps":
        require => [
            Exec["fetch-go"],
            Package["git-core"],
            Package["mercurial"],
            File["/tmp/blackforest-deps.bash"],
            File["$gopath/src/$importpath"],
        ],
        command => "/tmp/blackforest-deps.bash",
        environment => "GOPATH=$gopath",
    }

    file { "/home/$user/blackforest-reload":
        ensure => file,
        content => template("blackforest/blackforest-reload.bash"),
        mode => 755,
    }

    exec { "blackforest-install":
        require => Exec["blackforest-deps"],
        command => "/usr/local/go/bin/go install $importpath",
        environment => "GOPATH=$gopath",
        creates => "$gopath/bin/blackforest",
    }

    file { "/etc/init/blackforest.conf":
        mode => 600,
        owner => "root",
        group => "root",
        content => template("blackforest/blackforest.conf"),
    }

    exec { "blackforest-init":
        require => Exec["blackforest-install"],
        command => "$gopath/bin/blackforest init -catalog=\"$catalog\"",
        creates => $catalog,
        user => "root",
    }

    service { "blackforest":
        require => [File["/etc/init/blackforest.conf"], Exec["blackforest-init"]],
        ensure => running,
    }
}
