
# Houston API Config

Config can be provided as a YAML file when the API is started, e.g. `houston api --config my_config.yaml`, or by setting
the corresponding environment variable (shown in the table below).

| Field     | Type                                  | Description                                                                                                                                                                           | Environment Variable | Default | 
|-----------|---------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------|---------|
| password  | string                                | Admin password, which will be required for all admin routes (creating keys, deleting keys, etc.). If unset, the API will have no password protection, meaning anyone can create keys. | HOUSTON_PASSWORD     |         | 
| port      | string                                | Port from which to serve the API and dashboard.                                                                                                                                       | HOUSTON_PORT         | 8000    | 
| dashboard | [Dashboard Config](#dashboard-config) | Houston Dashboard config object. See below.                                                                                                                                           |                      |         |
| redis     | [Redis Config](#redis-config)         | Redis config object. See below.                                                                                                                                                       |                      |         | 


#### Dashboard Config

| Field   | Type   | Description                                                                                                                                            | Environment Variable  | Default | 
|---------|--------|--------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------|---------|
| enabled | bool   | If false, dashboard will be not be served.                                                                                                             | HOUSTON_DASHBOARD     | true    | 
| src     | string | Path to an HTML file to be served as the index page of the dashboard (for users who want a custom dashboard). If unset, the default dashboard is used. | HOUSTON_DASHBOARD_SRC |         | 


#### Redis Config

| Field    | Type   | Description                               | Environment Variable | Default        | 
|----------|--------|-------------------------------------------|----------------------|----------------|
| addr     | string | URL of Redis database to use.             | REDIS_ADDR           | localhost:6379 | 
| password | string | Password to use to access Redis database. | REDIS_PASSWORD       |                | 
| db       | int    | The Redis database number to use.         | REDIS_DB             | 0              | 


