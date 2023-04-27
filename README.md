# dispatch

A coredns plugin for dispatch request to other upstream by matched domain.

## Syntax

```
dispatch FROM... {
    to TO...
    
    healthcheck DURATION [no_rec]
    maxfails INTEGER
}
```

* `FROM...` is the file list which contains base domain to match for the request to be redirected. URL can also be used, currently only `HTTPS` is supported(due to security reasons).

  `.`(i.e. root zone) can be used solely to match all incoming requests as a fallback.

  Two kind of formats are supported currently:

    * `DOMAIN`, which the whole line is the domain name.

    * `server=/DOMAIN/...`, which is the format of `dnsmasq` config file, note that only the `DOMAIN` will be honored, other fields will be simply discarded.

  Text after `#` character will be treated as comment.

  Unparsable lines(including whitespace-only line) are therefore just ignored.

* `to TO...` are the destination endpoints to redirected to. This is a mandatory option.

  The `to` syntax allows you to specify a protocol, a port, etc:

  `[dns://]IP[:PORT]` use protocol specified in incoming DNS requests, it may `UDP` or `TCP`.

  `[udp://]IP:[:PORT]` use `UDP` protocol for DNS query, even if request comes in `TCP`.

  `[tcp://]IP:[:PORT]` use `TCP` protocol for DNS query, even if request comes in `UDP`.

  `tls://IP[:PORT][@TLS_SERVER_NAME]` for DNS over TLS, if you combine `:` and `@`, `@` must come last. Be aware of some DoT servers require TLS server name as a mandatory option.

  `json-doh://URL` use [JSON](https://developers.google.com/speed/public-dns/docs/doh/json) `DNS over HTTPS` for DNS query.

  `ietf-doh://URL` use IETF([RFC 8484](https://tools.ietf.org/html/rfc8484)) `DNS over HTTPS` for DNS query.

  `doh://URL` randomly choose JSON or IETF `DNS over HTTPS` for DNS query, make sure the upstream host support both of type.

  Example:

    ```
    dns://1.1.1.1
    8.8.8.8
    tcp://9.9.9.9
    udp://2606:4700:4700::1111

    tls://1.1.1.1@one.one.one.one
    tls://8.8.8.8
    tls://dns.quad9.net

    doh://cloudflare-dns.com/dns-query
    json-doh://1.1.1.1/dns-query
    json-doh://dns.google/resolve
    ietf-doh://dns.quad9.net/dns-query
    ```

## Example
```
.:53 {
    dispatch https://expample.com/dnsmasq-dns.conf custom-domain.conf {
        to 1.1.1.1 tls://1.1.1.1@one.one.one.one
        
        healthcheck 10
        maxfails 5
    }
    dispatch home.conf {
        to 10.1.1.10:53
    }
    forward . 8.8.8.8 
}

```