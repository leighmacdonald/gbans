# Network Tools

A set of IP/network utilities for operators. All are available under **Admin** to users
with the `network_read` permission.

## Find Players By IP

`/admin/network/playersbyip` — enter an IP (or CIDR) to list every player seen connecting
from it, with server, time, name, SteamID, and ASN. Useful for spotting evasion or
shared-IP groups.

## IP Info

`/admin/network/ipinfo` — look up a single IP to see its geolocation (country, region,
city, and coordinates on a map) and its ASN/ISP. Relies on the
[IP2Location](../install/config.md#ip2location) database.

## CIDR / IP Blocklist

`/admin/network/cidrblocks` — manage the CIDR network blocks gbans matches against, along
with IP and SteamID whitelists. Whitelisted IPs and players are exempt from IP-based bans.
These blocklist sources and whitelists are shared with the [ban](bans.md) system.
