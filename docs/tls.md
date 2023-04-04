
# Transport Layer Security (TLS) / Secure Socket Layer (SSL) / HTTPS

To configure TLS/SSL for a Houston instance, and allow communication to and from the server via HTTPS, 
you can either:
1. Configure your Houston server to automatically generate/renew certificates via the [ACME protocol](https://en.wikipedia.org/wiki/Automatic_Certificate_Management_Environment) and [Let's Encrypt](https://letsencrypt.org/)
2. Provide your own certificate

## Automatic TLS/SSL Certificate Generation and Renewal

Houston can use go's [acme.autocert](https://golang.org/x/crypto/acme/autocert) library to automatically generate a 
certificate if a domain name is provided in the [config](./config.md#tls-config).

The steps to configure this are as follows:

1. Acquire a domain name or subdomain
2. Set `config.TLS.Host` to the value of your host name, e.g. 'houston.example.com'. See [TLS Config](./config.md#tls-config) for alternative ways to set this value. You must also provide a password for your server when using TLS (`config.Password`).
3. Start a houston server (`houston api`). 
4. Determine the (static) IP address of your Houston server
5. Point the domain name to your server with DNS records (described below)

Example YAML config for auto TLS:

```yaml
password: changeme
tls:
  host: houston.example.com
```

Example environment variable config for auto TLS:

```bash
export HOUSTON_PASSWORD="changeme"
export TLS_HOST="houston.example.com"
```


## Provide Your Own TLS/SSL Certificate

If you want to provide your own certificate:
1. Acquire a domain name for your server
2. Point the domain name to your server (described below) with DNS configuration
3. Generate a certificate
4. Upload the certificate to the server so that `houston` can find them locally  
5. Provide the SSL certificate when starting the Houston server (configuration described in [TLS Config](./config.md#tls-config)). Ensure that 'auto' is set to `false`.

## Example DNS Record

Once you have acquired a domain name, point the domain to the IP address of your
Houston server using DNS records, for example:

    Name  houston.example.com
    Type  A
    TTL   300
    Data  12.34.156.178

From then on, your server's base URL will become `https://houston.example.com/api/v1`, i.e. `https://<your domain name>/api/v1`.

