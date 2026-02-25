# Steam Datagram Relay (SDR)

:::info

Steam Datagram Relay (SDR) is Valve's virtual private gaming network. Using our APIs, you can not only carry your game
traffic over the Valve backbone that is dedicated for game content, you also gain access to our network of relays.
Relaying the traffic protects your servers and players from DoS attack, because IP addresses are never revealed. All
traffic you receive is authenticated, encrypted, and rate-limited. Furthermore, for a surprisingly high number of
players, we can also find a faster route through our network, which actually improves player ping times.

- Valve

:::

gbans provides a few features to support [SDR](https://partner.steamgames.com/doc/features/multiplayer/steamdatagramrelay)
functionality:

- Fetching the correct FakeIP address. When updating the status via rcon, if the fake sdr ip has changed, it will be updated.
  - If enabled, there is support for automatically updating DNS records as well via cloudflare API. No other providers
    are currently supported.

:::danger

When you enable SDR on your servers, you lose the ability to do any sort of IP / CIDR / ASN based bans. You are limited
to steamid bans since the clients are also given dynamic fake SDR addresses as well. The pros and cons of this should be
weighed by your administrator before changing these settings.

:::
