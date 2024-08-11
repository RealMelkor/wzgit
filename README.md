# wzgit

A minimalist software forge

## Features

* Allow users to create and manage git repositories
* Private and public repositories
* Serving git repositories on the http and ssh protocols
* LDAP authentication
* 2FA with time-based one-time passwords
* Option to use token authentication when doing git operations
* Basic brute-force protection
* User groups
* Privilege system for read/write access
* Support for sqlite and mysql databases
* No javascript

## Requirements

* go (at compile-time)
* git (at run-time)

### Fedora, RedHat
```
dnf install git golang
```

### Debian, Ubuntu
```
apt install git golang
```

## Setup

* Build the program with the following command :
```
go build
```
* Copy config.yaml into either /etc/wzgit, /usr/local/etc/wzgit or the working directory
* Edit the configuration file as you want
* Execute wzgit

On Linux, wzgit can be run as a systemd service :
```
adduser -d /var/lib/wzgit -m -U wzgit
cp ./service/wzgit.service /etc/systemd/system/
go build
cp ./wzgit /usr/bin/wzgit
mkdir /etc/wzgit
cp ./config.yaml /etc/wzgit/
chown -R wzgit:wzgit /var/lib/wzgit
systemctl enable --now wzgit
```

## Demo

You can try a [public instance of wzgit][0]

## Contact

For inquiries about this software or the instance running at https://wz.rmf-dev.com, you can contact the main maintainer of this project at : inquiry@rmf-dev.com

[0]: https://wz.rmf-dev.com
