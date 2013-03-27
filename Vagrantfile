# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "precise32"
  config.vm.box_url = "http://files.vagrantup.com/precise32.box"
  config.vm.provision :puppet do |puppet|
      puppet.manifests_path = "misc/vagrant/puppet/manifests"
      puppet.module_path = "misc/vagrant/puppet/modules"
      puppet.manifest_file = "default.pp"
  end
  config.vm.network :forwarded_port, guest: 10710, host: 10710
end
