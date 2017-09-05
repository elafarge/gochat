# Gochat

## Load test results

Ran against our dev. env. (see below) on decent Core-i7 (quad core @2.7GHz):
 * 1 user, 40000 msg/s (4MiB/s)
 * 5000 websocket connections in parrallel, 9msg/s (10MiB/s)

## Getting started

#### Requirements
 * [docker](https://docs.docker.com/engine/installation/)
 * [make](https://www.gnu.org/software/make/)
 * `docker-compose` (usually bundled with `docker`, otherwise `pip install
   docker-compose`)


#### (Re)starting the dev. env.
Make sure you have ports `4690` and `4691` available on your machine and simply
run:
```shell
make dev
```
This should make our frontend available under `http://localhost:4690` and our
backend under `http://localhost:4691`.

##### Usage
Just visit the frontend under `http://localhost:4690/#coop` to log into the
chat, as user `coop` and, in another window, as user `gordon` by reaching
`http://localhost:4690/#gordon`.

Using the above handles (without the `#` sign), your users shoud be able to
communicate. Feel free to add a third `#ben` user... and as many other users
with as many names as you wish.

##### Notes

The command can be used to spawn the dev. env. for the first time but, thanks to
the magic of `make`, it can also be used to repull the dependencies, rebuild the
golang binary (and its associated container) and restart it. In other words, the
development workflow boils down to

" change code -> `make dev` -> fix bugs -> `make dev` -> ... "

Note that `make` decides to re-pull the dependencies and rebuild the binary only
if necessary (according to the files that changed).

#### Monitoring your dev. env.
Since components are launched in containers, I highly recommand to use [ctop](https://github.com/bcicen/ctop).

#### Tailing logs from the dev. env.
```shell
make devlog
```

#### Running load tests against the dev. env.
Under `load-test`, we ship a Go program that can be used to emulate a WebSocket
client and send/receive messages. First of all, you'll need to issue some sysctl
commands (as root) to tune the Kernel and get the best out of `gochat`:

```shell
sysctl -w fs.file-max=11000000
sysctl -w fs.nr_open=11000000
ulimit -n 11000000
sysctl -w net.ipv4.tcp_mem="100000000 100000000 100000000"
sysctl -w net.core.somaxconn=10000
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w fs.file-max=11000000
sysctl -w fs.nr_open=11000000
sysctl -w net.ipv4.tcp_mem="100000000 100000000 100000000"
sysctl -w net.core.somaxconn=10000
```

You may have to increase the `ulimit` corresponding to the number of opened file
descriptor that can be opened depending on the amount of Websocket connexions to
be handled in parrallel.

Now, it's time to build and run the load-testing program:

```shell
# 1 simultaneous user, send 100 msgs per second to himself
go build -o lt . && ./lt -user-count 1 -rate 100
```

#### Destroying the dev. env.
```shell
make devdown
```

#### Cleaning build artifacts & vendor folder
```shell
make clean
```
(it also destroys the dev. env. if the latter is up)

#### Without docker and docker-compose
You'll need Go 1.8+ and `glide` (`go get github.com/Masterminds/glide`).

* Pull the dependencies with `glide install`
* Build gochat in `./main`
* Run it (`./gochat -h` for more CLI help)

Here's an example of a working build command, that would also start `gochat` on
port `4691` with verbose logs:

```shell
go build -o gochat ./main && ./gochat -log-level debug
```

You'll additionnaly need to run the frontend, which can simply be opened in your
browser, as a static file (it lives under `./front`).

Author
------
 * Étienne Lafarge <etienne.lafarge _at_ gmail.com>
