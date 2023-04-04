
# Houston API Config

Config can be provided as a YAML file when the API is started, e.g. `houston api --config my_config.yaml`, or by setting
the corresponding environment variable (shown in the table below).

| Field     | Type                                  | Description                                                                                                                                                                           | Environment Variable | Default | 
|-----------|---------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------|---------|
| password  | string                                | Admin password, which will be required for all admin routes (creating keys, deleting keys, etc.). If unset, the API will have no password protection, meaning anyone can create keys. | HOUSTON_PASSWORD     |         | 
| port      | string                                | Port from which to serve the API and dashboard. This is ignored if [TLS Config](#tls-config) is provided; all traffic will be served on port 443.                                     | HOUSTON_PORT         | 8000    | 
| dashboard | [Dashboard Config](#dashboard-config) | Houston Dashboard config object. See below.                                                                                                                                           |                      |         |
| redis     | [Redis Config](#redis-config)         | Redis config object. See below.                                                                                                                                                       |                      |         | 
| tls       | [TLS Config](#tls-config)             | Transport Layer Security (TLS) / SSL config object. See below.                                                                                                                        |                      |         | 


#### Dashboard Config

| Field   | Type   | Description                                                                                                                                                                                          | Environment Variable  | Default | 
|---------|--------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------|---------|
| enabled | bool   | If false, dashboard will be not be served.                                                                                                                                                           | HOUSTON_DASHBOARD     | true    | 
| src     | string | Path to an HTML file to be served as the index page of the dashboard (for users who want a custom dashboard). If unset, the [default dashboard](https://github.com/datasparq-ai/houston-ui) is used. | HOUSTON_DASHBOARD_SRC |         | 


#### Redis Config

| Field    | Type   | Description                               | Environment Variable | Default        | 
|----------|--------|-------------------------------------------|----------------------|----------------|
| addr     | string | URL of Redis database to use.             | REDIS_ADDR           | localhost:6379 | 
| password | string | Password to use to access Redis database. | REDIS_PASSWORD       |                | 
| db       | int    | The Redis database number to use.         | REDIS_DB             | 0              | 


#### TLS Config

Transport Layer Security (TLS) / SSL configuration. 

Houston will automatically use HTTPS if 'host' is provided, or if certificate is found at the path provided (or the default path). 
See [TLS/SSL/HTTPS](./tls.md) for a guide on configuring TLS and how 'auto' works. 

| Field    | Type   | Description                                                                                                                                                                                            | Environment Variable | Default  |
|----------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------|----------|
| auto     | bool   | If true, TLS certificates will be generated automatically using the host provided. Note that this is enabled by default, but TLS is not used if no host is provided.                                   | TLS_AUTO             | true     |
| host     | string | The hostname that will be used for this Houston server, e.g. 'houston.example.com'. This must be provided in order to use automatic TLS configuration.                                                 | TLS_HOST             |          |
| certFile | string | TLS/SSL certificate file path. If the certificate is signed by a certificate authority, the file should be the concatenation of the server's certificate, any intermediates, and the CA's certificate. | TLS_CERT_FILE        | cert.pem |
| keyFile  | string | TLS/SSL certificate private key file path. Note that this certificate will not be used unless 'auto' is also set to false.                                                                             | TLS_KEY_FILE         | key.pem  |


## Full Example

Below is an example config.yaml with every field specified. 

```yaml
password: changeme
port: 8000  # note: this is ignored if TLS config is provided
dashboard: 
  enabled: true
  src: 'my-dashboard.html'
redis:
  addr: 'redis.example.com:6379'
  password: changeme
  db: 0
tls:
  auto: false
  host: 'houston.example.com'
  certFile: 'houston.example.com.crt'
  keyFile: 'houston.example.com.key'
```
