
# Database Schema

### Redis Database

The Redis database schema is shown below.
It is recommended to have one database per organisation to minimise costs, and one API key per project/environment.

Redis Schema:

```yaml
<api key>:                           # stored as json string, made as small as possible
  n: "Moon Mission"                    # key/project/env name
  u: 234234                            # usage (number of requests made to this key)
<api key>|<mission id>:              # stored as json string, made as small as possible
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
<key>|p|<plan-name>: "{\"name\": \"apollo\", \"stages\": [] }"
```

### Local Database

The local database schema is shown below. The local db behaves exactly the same as the redis db, 
but uses a go's `sync.Mutex` to stay transactional. 
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
  
  plan|<plan-name>: "{\"name\": \"apollo\", \"stages\": [] }"

```

### FAQ

Q: Why store plan params and services in the mission as well as the plan?
A: This allows services to only download the mission to execute each stage, which prevents any errors from occurring when plans are updated mid-mission.
