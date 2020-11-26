# HTTPD Configs

## Caddy w/cloudflare

    gbans.uncledane.com {
        reverse_proxy /* 192.168.0.222:6006
        encode gzip
        tls your@email.com {
            dns cloudfalre your_api_token
        }
    }