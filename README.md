# Harbour: An agent for containers
![Harbour](doc/pic/harbour.jpg "Harbour")

## What is Harbour
Harbour is a container node agent, which runs on the host machine, works as a proxy for users to eliminate the differences between containers. For example, harbour takes over docker local socket and network port to provides services for clients.

## How it works
Harbour is implemented as a configurable, pluggable HTTP proxy for the Docker API. As a proxy, harbour stands between user and container engine. When receives command from user, harbour identify and dispatch it to corresponding container engine in backend, such as docker or rkt. At present, you can use docker client as a user client to communicate with harbour, and in the long run, harbour client will be provided for the convenience of customers. 

## Current Status
At the time of writing, harbour can work as a proxy for docker daemon, support for other containers(e.g. rkt) is already under way. It's a work in progress, so bear with us :)

## Try it out
Although harbour is still in development, we encourage you to try out the tool and give feedback. 

### Build

Installation is simple as:

	go get github.com/huawei-openlab/harbour

or as involved as:

	git clone https://github.com/huawei-openlab/harbour.git
	cd harbour
	go get -d ./...
	go build
	
## Usage

```
$  ./harbour
Usage: harbour [OPTIONS] [arg...]

Options:

  --container-runtime=docker                 Container runtime to choose
  -D, --debug=false                          Enable debug mode
  -d, --daemon=false                         Enable daemon mode
  --docker-sock=/var/run/docker-real.sock    Path to docker sock file
  -G, --group=docker                         Group for the unix socket
  -H, --host=[]                              Daemon socket(s) to connect to
  -h, --help=false                           Print usage
  -v, --version=false                        Print version information and quit

Commands:

Run 'harbour COMMAND --help' for more information on a command.

```
### Default mode
Make sure docker located in `PATH`, run docker daemon, and do the following work:
- If binary used, run docker daemon with `-H unix:///var/run/docker-real.sock`
- If ubuntu lxc-docker used, open file `/etc/default/docker`, add `-H unix:///var/run/docker-real.sock` in `DOCKER_OPTS`, save and restart docker.
- If systemd used to manage docker service, open the service file corresponding to docker, add `-H unix:///var/run/docker-real.sock`, save and `systemctl restart docker`.

Then run `harbour -d D` using root(Listen to `/var/run/docker.sock` and forward it to `/var/run/docker-real.sock` by default)

### User-defined mode
`harbour -d -D --docker-sock=/var/run/dockerxxx.sock`(specified sock for docker) `-H unix:///a/b/c.sock`(specified sock for harbour)  `-H tcp://:4567`(specified tcp port for harbour)

### Examples

#### Proxy for Docker
Harbour works as a proxy of docker by default. Below is an example of harbour working for docker in default mode:
- Run harbour daemon
```
$ ./harbour -d -D &
[1] 15127
root@ubuntu:~/Applications/Go/src/github.com/huawei-openlab/harbour# DEBU[0000] trap init...
DEBU[0000] Registering GET,
DEBU[0000] Registering POST,
DEBU[0000] Registering DELETE,
DEBU[0000] Listening for HTTP on unix (/var/run/docker.sock)
DEBU[0000] docker group found. gid: 999
```
- Run docker daemon with parameters
```
$ docker -d -H unix:///var/run/docker-real.sock &
[2] 15131

INFO[0000] Listening for HTTP on unix (/var/run/docker-real.sock)
INFO[0000] [graphdriver] using prior storage driver "aufs"
INFO[0000] Loading containers: start.
....................
INFO[0000] Loading containers: done.
INFO[0000] Daemon has completed initialization
INFO[0000] Docker daemon                                 commit=786b29d execdriver=native-0.2 graphdriver=aufs version=1.7.1
```
- Run docker client
```
$ docker images
DEBU[0512] Calling GET
DEBU[0512] Request get: &{GET /v1.19/images/json HTTP/1.1 1 1 map[User-Agent:[Docker-Client/1.7.1]] 0x8f5e30 0 [] false /var/run/docker.sock map[] map[] <nil> map[] @ /v1.19/images/json <nil>}
DEBU[0512] Request's url: /v1.19/images/json
INFO[0292] GET /v1.19/images/json
REPOSITORY          TAG                 IMAGE ID            CREATED             VIRTUAL SIZE
busybox             latest              17583c7dd0da        8 days ago          1.109 MB
ubuntu              latest              a5a467fddcb8        2 weeks ago         187.9 MB
<none>              <none>              8c2e06607696        6 months ago        2.433 MB
```

#### Proxy for rkt
If you want harbour to work as a proxy for rkt, `--container-runtime=rkt` option can be added when harbour daemon started. Example illustrated as below:

- Run harbour daemon with rkt parameter

```
$ ./harbour --container-runtime=rkt -d -D &
[3] 25594
root@ubuntu:~/Applications/Go/src/github.com/huawei-openlab/harbour# DEBU[0000] trap init...
DEBU[0000] Registering POST,
DEBU[0000] Registering DELETE,
DEBU[0000] Registering GET,
DEBU[0000] Listening for HTTP on unix (/var/run/docker.sock)
DEBU[0000] docker group found. gid: 999

```

- Docker cmd will be converted to rkt operations

```
$ docker images
DEBU[0222] Calling GET
DEBU[0222] Request get: &{GET /v1.19/images/json HTTP/1.1 1 1 map[User-Agent:[Docker-Client/1.7.1]] 0x8f7e30 0 [] false /var/run/docker.sock map[] map[] <nil> map[] @ /v1.19/images/json <nil>}
DEBU[0222] Request's url :/v1.19/images/json
DEBU[0222] Request's url path: /v1.19/images/json
DEBU[0222] Transforwarding request body:
KEY                                                                     APPNAME                         IMPORTTIME                              LATEST
sha512-ca0bee4ecb888d10cf0816ebe7e16499230ab349bd3126976ab60b9b1db2e120 coreos.com/rkt/stage1:0.8.0     2015-09-15 17:38:15.068 +0800 CST       false
sha512-b56f0c5d3808771c5571c1d98629eab0ec02dfe71910ff57c13a32a1497f1add example:0.0.1                   2015-09-15 17:39:17.747 +0800 CST       false
sha512-f7120d3a61fd72c746b3550a4d2ae55f0119c8086d5f1e6afe8310fe2cc4f4aa example:0.0.1                   2015-09-15 17:50:08.142 +0800 CST       false
sha512-6f7cd1c85308e01c3e9e628804c6c01510c3f4895ff1674d9ece91d4bf87874d example:0.0.1                   2015-09-15 17:58:31.36 +0800 CST        false
sha512-077395c1a7acf9543f202f1182540bad42d452127d0f57baae30a3e3e8b2f5cb example:0.0.1                   2015-09-15 19:24:08.305 +0800 CST       false
sha512-13885d66415514cdf908a033a7d2025c9cdf5fed2336fa7ea2e8821e440de9cb example:0.0.1                   2015-09-16 14:29:25.522 +0800 CST       false
sha512-e3369b474208cd5d8aa40d04a6c609d5feace3c56501cbabed2b847db93941ef example:0.0.1                   2015-09-16 14:38:35.04 +0800 CST        false

```

## How to involve
If any issues are encountered while using the harbour project, several avenues are available for support:
<table>
<tr>
	<th align="left">
	Issue Tracker
	</th>
	<td>
	https://github.com/huawei-openlab/harbour/issues
	</td>
</tr>
<tr>
	<th align="left">
	Google Groups
	</th>
	<td>
	https://groups.google.com/forum/#!forum/harbour-dev
	</td>
</tr>
</table>


## Who should join
- Container(docker,rkt,lxc) developer/user
- Ones who want to use container but do not care underlying runtime.

## Certificate of Origin
By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. 

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
660 York Street, Suite 102,
San Francisco, CA 94110 USA

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Format of the Commit Message

You just add a line to every git commit message, like this:

    Signed-off-by: Meaglith Ma <maquanyi@huawei.com>

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.
