# Khulnasoft Tunnel client

Contains the command-line client for Khulnasoft Tunnel, a tunneling daemon that proxies traffic from the Khulnasoft network to your origins.
This daemon sits between Khulnasoft network and your origin (e.g. a webserver). Khulnasoft attracts client requests and sends them to you
via this daemon, without requiring you to poke holes on your firewall --- your origin can remain as closed as possible.
Extensive documentation can be found in the [Khulnasoft Tunnel section](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps) of the Khulnasoft Docs.
All usages related with proxying to your origins are available under `tunnellink tunnel help`.

You can also use `tunnellink` to access Tunnel origins (that are protected with `tunnellink tunnel`) for TCP traffic
at Layer 4 (i.e., not HTTP/websocket), which is relevant for use cases such as SSH, RDP, etc.
Such usages are available under `tunnellink access help`.

You can instead use [WARP client](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/configuration/private-networks)
to access private origins behind Tunnels for Layer 4 traffic without requiring `tunnellink access` commands on the client side.


## Before you get started

Before you use Khulnasoft Tunnel, you'll need to complete a few steps in the Khulnasoft dashboard: you need to add a
website to your Khulnasoft account. Note that today it is possible to use Tunnel without a website (e.g. for private
routing), but for legacy reasons this requirement is still necessary:
1. [Add a website to Khulnasoft](https://support.khulnasoft.com/hc/en-us/articles/201720164-Creating-a-Khulnasoft-account-and-adding-a-website)
2. [Change your domain nameservers to Khulnasoft](https://support.khulnasoft.com/hc/en-us/articles/205195708)


## Installing `tunnellink`

Downloads are available as standalone binaries, a Docker image, and Debian, RPM, and Homebrew packages. You can also find releases [here](https://github.com/khulnasoft/tunnellink/releases) on the `tunnellink` GitHub repository.

* You can [install on macOS](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/install-and-setup/installation#macos) via Homebrew or by downloading the [latest Darwin amd64 release](https://github.com/khulnasoft/tunnellink/releases)
* Binaries, Debian, and RPM packages for Linux [can be found here](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/install-and-setup/installation#linux)
* A Docker image of `tunnellink` is [available on DockerHub](https://hub.docker.com/r/khulnasoft/tunnellink)
* You can install on Windows machines with the [steps here](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/install-and-setup/installation#windows)
* To build from source, first you need to download the go toolchain by running `./.teamcity/install-khulnasoft-go.sh` and follow the output. Then you can run `make tunnellink`

User documentation for Khulnasoft Tunnel can be found at https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps


## Creating Tunnels and routing traffic

Once installed, you can authenticate `tunnellink` into your Khulnasoft account and begin creating Tunnels to serve traffic to your origins.

* Create a Tunnel with [these instructions](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/create-tunnel)
* Route traffic to that Tunnel:
  * Via public [DNS records in Khulnasoft](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/routing-to-tunnel/dns)
  * Or via a public hostname guided by a [Khulnasoft Load Balancer](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-apps/routing-to-tunnel/lb)
  * Or from [WARP client private traffic](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-networks/private-net/)


## TryKhulnasoft

Want to test Khulnasoft Tunnel before adding a website to Khulnasoft? You can do so with TryKhulnasoft using the documentation [available here](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-networks/do-more-with-tunnels/trykhulnasoft/).

## Deprecated versions

Khulnasoft currently supports versions of tunnellink that are **within one year** of the most recent release. Breaking changes unrelated to feature availability may be introduced that will impact versions released more than one year ago. You can read more about upgrading tunnellink in our [developer documentation](https://developers.khulnasoft.com/khulnasoft-one/connections/connect-networks/downloads/#updating-tunnellink).

For example, as of January 2023 Khulnasoft will support tunnellink version 2023.1.1 to tunnellink 2022.1.1.
