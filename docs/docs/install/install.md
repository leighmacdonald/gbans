---
sidebar_position: 1
---

# Installation Guide

Basic installation overview of the gbans server and sourcemod plugin.



## System & Performance Considerations

Gbans is lightweight and can handle a small to moderately sized community with a dual-core CPU and 4GB of memory.

Special considerations need to be made when using extended functionality:

It's recommended, but not required, to use a dedicated server for installation. The application is fairly lightweight,
but some features
will take some computational power which can interrupt other processes. This includes things like downloading demos
over SSH/SCP and processing users stats. If you omit these features, it should be able to run alongside a game server
fairly
well on a VPS. Ram usage is pretty negligible, but if you have a lot of servers and a long history, you may want to
increase the ram allocated to postgres.

If you are hosting game servers and gbans on the same host, you will likely want to
specify [GOMAXPROCS](https://pkg.go.dev/runtime#hdr-Environment_Variables)
when starting gbans. This allows you to set processor affinity/cpuset properly to ensure they are not fighting each other
for resources.

:::note

There is currently a [golang issue](https://github.com/golang/go/issues/33803) related to docker containers
and setting `GOMAXPROCS` properly.

:::

[IP2Location](https://lite.ip2location.com/) updates are a fairly intensive process, so considerations should be taken as far as how and when to update
the database
to ensure it doesn't impact other things on the system.

## Runtime requirements

Running the binaries is very easy as they are statically compiled. All frontend assets are embedded into the binary
to make deployment as trivial as possible.

- Any modern-ish postgresql install with [PostGIS](https://postgis.net/) & [ip4r](https://github.com/RhodiumToad/ip4r)
  extensions. All non-EOL versions of postgres should work.
- A platform that go supports. Only linux and windows amd64 are tested, but as far as I know, others should work.

## Sourcemod Plugins

The following extensions must be installed for gbans to work, see their documentation for up-to-date installation
instructions:

- [sm-ripext](https://github.com/ErikMinekus/sm-ripext) Provides HTTP(S) client functionality
- [sm-json](https://github.com/clugg/sm-json) `Required for development only` Provides JSON encoding/decoding.
- [Connect](https://github.com/asherkin/connect) Provides `OnClientPreConnectEx`
- [SourceTVManager](https://github.com/peace-maker/sourcetvmanager) Interface to interact with the SourceTV server from
  SourcePawn.

## gbans Server

### Compile from source

Precompiled binaries will be provided once the project is in a more stable state. It's recommended to use the docker
images as they are currently the only tested usecase.

- [make](https://www.gnu.org/software/make/) Not strictly required but provides predefined build commands
- [golang 1.24+](https://golang.org/) Version >=1.23 is required.
- [PostgreSQL](https://www.postgresql.org/) Version 16 is the only version currently tested against. All non-EOL
  versions should be supported.
    - [PostGIS](https://postgis.net/) Provides some basic GIS functionality.
    - [ip4r](https://github.com/RhodiumToad/ip4r) Improved ip/cidr indexed and types.
- [Node.js >=18.17.1](https://nodejs.org/en/) To build frontend
    - [pnpm](https://pnpm.io/) JS package manager
- [Sourcemod 1.12](https://www.sourcemod.net/) - Sourcemod installation

Basic steps to build the binary packages:

If you do not already have sourcemod, you can download and extract sourcemod to a directory of your choosing with the
following:

```shell
mkdir -p ~/sourcemod &&  wget https://sm.alliedmods.net/smdrop/1.12/sourcemod-1.12.0-git7110-linux.tar.gz -O ~/sourcemod/sm.tar.gz && tar xvfz ~/sourcemod/sm.tar.gz -C ~/sourcemod
```

Clone the gbans repository

```shell
git clone git@github.com:leighmacdonald/gbans.git && cd gbans
````

Build the projects, replace SM_ROOT with the path to your sourcemod installation directory (the folder with addons and
cfg folders inside).

```shell
SM_ROOT=~/sourcemod make
````

You should now have a binary located at `./build/$platform/gbans`

### Docker

```shell
docker run -d --restart unless-stopped \
    -p 6006:6006 \
    --dns=1.1.1.1 \
    -v /home/ubuntu/gbans/gbans.yml:/app/gbans.yml:ro \
    --name gbans \
    ghcr.io/leighmacdonald/gbans:latest
```

Substitute `/home/ubuntu/gbans/gbans.yml` with the location of your config.

This configuration will restart gbans unless explicitly stopped, and names the container for easy log access/stopping.

Note that docker defaults to 64MB shm which eventually becomes problematic once data exceeds the limits. If queries
suddenly start to not return results, you probably need to increase this.

You can add `shm_size: 512m` to the docker compose file for postgres, or `--shm-size=512m` if running docker command
directly.

## Reverse Proxy

### Caddy w/cloudflare

```
{
        acme_ca https://acme.zerossl.com/v2/DV90
        email yours@example.com
}

example.com {
    # https://www.cloudflare.com/ips/
    @cloudflare {
        remote_ip 173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/13 104.24.0.0/14 172.64.0.0/13 131.0.72.0/22 2400:cb00::/32 2606:4700::/32 2803:f800::/32 2405:b500::/32 2405:8100::/32 2a06:98c0::/29 2c0f:f248::/32
    }

    tls {
            dns cloudflare <token>
    }

    # Process requests from Cloudflare IPs
    handle @cloudflare {
        reverse_proxy localhost:8080 {
            # Sets X-Forwarded-For as the value Cloudflare gives us for CF-Connecting-IP.
            header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}
        }
    }

    # Deny requests from non-Cloudflare IPs
    handle {
        respond "Access Denied" 403
    }
}

```

### Apache 2.4

Be sure to run `sudo a2enmod proxy_http ssl` first.

```
<IfModule mod_ssl.c>
<VirtualHost *:443>
        ServerName example.com

        ProxyPass / http://127.0.0.1:6006/
        ProxyPassReverse / http://127.0.0.1:6006/

        ServerAdmin your@email.com

        #Can be disabled if wanted
        ErrorLog ${APACHE_LOG_DIR}/error.log
        CustomLog ${APACHE_LOG_DIR}/access.log combined

SSLCertificateFile /etc/cloudflare/example.com.pem
SSLCertificateKeyFile /etc/cloudflare/example.com.key
</VirtualHost>
</IfModule>
```

If using Cloudflare to provide user location, you can use Origin Certificates to generate a long-lasting SSL
certificate.
