
# Database Schema

### Redis Database

The Redis database schema is shown below.
It is recommended to have one database per organisation to minimise costs, and one API key per project/environment.

Redis Schema:

```yaml
<api key>|n: "Moon Mission"          # key/project/env name
<api key>|u: 3                       # usage (number of requests made to this key)
<api key>|c: m1,m2                   # completed, stored as JSON string, list of completed mission IDs (strings)
<api key>|p|<plan-name>:             # plan, stored as JSON string, identical to mission without stage timings
  name: "apollo"                       # plan name
  stages: []                           # list of stages
<api key>|a|<plan-name>: m1,m2,m3    # active, list of active mission IDs (strings) for a plan
<api key>|<mission id>:              # mission, stored as json string, made as small as possible
  n: apollo                            # name (plan name)
  i: <mission_id>                      # id
  s:                                   # stages
   - n: foo                              # name
     a: my-service                       # service
     p:                                  # params
       foo: bar                          
       bar: foo
     d: [foo2]                           # downstream 
     u: [foo0]                           # upstream
     s: 1                                # state
     t: 2022-03-03T16:35:47.559127Z      # start
     e: 2022-03-03T16:35:47.559127Z      # end
     x: 53                               # x position in UI (concept)
     y: 12                               # y position in UI
  a:                                   # services
   - n: my-service                       # name
     t:                                    # trigger
       m: pub/sub                            # method
  t: 2022-03-03T16:35:47.559127Z       # start
  e: 2022-03-03T16:35:47.559127Z       # end
  p:                                   # params (plan params + mission params)
    foo: bar
m|p: <hash>                          # server metadata - hashed password
m|s: <random string>                 # salt
```

### Local Database

The local database schema is shown below. The local db behaves exactly the same as the redis db, 
but uses go's `sync.Mutex` to stay transactional. 
This may cause it to perform slower than the redis db for large plans or multiple users. 

Simple schema used by local db, which is of type `map[string]map[string]string`:

```yaml
<api key>:                            # (the api key) [hash]
  n: "Moon Mission"                     # key/project/env name
  u: 234234                             # usage (number of requests made to this key)
  <mission id>:                          # stored as json string, made as small as possible
    n: apollo                            # name (plan name)
    i: <mission_id>                      # id
    s:                                   # stages
     - n: foo                              # name
       a: my-service                       # service
       p:                                  # params
         foo: bar
         bar: foo
       d: [foo2]                           # downstream 
       u: [foo0]                           # upstream
       s: 1                                # state
       t: 2022-03-03T16:35:47.559127Z      # start
       e: 2022-03-03T16:35:47.559127Z      # end
       x: 53                               # x position in UI (concept)
       y: 12                               # y position in UI
    a:                                   # services
     - n: my-service                       # name
       t:                                    # trigger
         m: pub/sub                            # method
    t: 2022-03-03T16:35:47.559127Z       # start
    e: 2022-03-03T16:35:47.559127Z       # end
    p:                                   # params (mission params)
      foo: bar
  p|<plan-name>: "{\"name\": \"apollo\", \"stages\": [] }"
  a|<plan-name>: m1,m2,m3
m|p: <hash>                          # server metadata - hashed password
m|s: <random string>                 # salt
```

### FAQ

Q: Why store plan params and services in the mission as well as the plan?
A: This allows services to only download the mission to execute each stage, which prevents any errors from occurring when plans are updated mid-mission.
