# Development Environment Setup

## Linters & Formatting

`make fmt` Format go code with go fmt

`make lint` will run the golangci-lint tool

## Updating Dependencies

`make bump_deps` Can be used to update both go and js/ts libraries.

## Seeding data

There is a simple `seed` command you can use to pre-load some data for working with while
developing.

    {
        "settings": {
            "rcon": "common_rcon_password"
        },
        "admins": [ "76561198084134025" ],
        "players": [
            "76561198080000000",
            "76561198080000001",
        ],
        "servers": [
          {
            "short_name": "test-1",
            "host": "test-1.us.server.com",
            "password": "xxxxxxxxxxxxxxxxxxxx",
            "location": [ 58.377956, 24.897070 ],
            "enabled": false,
            "region": "na",
            "cc": "us"
          },
          {
            "short_name": "test-2",
            "host": "test-2.us.server.com",
            "password": "xxxxxxxxxxxxxxxxxxxx",
            "location": [ 58.377956, 24.897070 ],
            "enabled": false,
            "region": "na",
            "cc": "us"
          },
        ]
    }

You can then import the data with:

`./gbans seed -f seed.json -r common_rcon_password`

