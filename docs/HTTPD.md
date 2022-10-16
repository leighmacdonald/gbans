# HTTPD Configs

## Caddy w/cloudflare

    example.com {
        reverse_proxy /* internal_host:6006
        encode gzip
        tls your@email.com {
            dns cloudfalre your_api_token
        }
    }
