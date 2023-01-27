
# Houston Websocket

The Houston API serves a websocket from /ws and sends event data to all connected clients to enable real-time
mission tracking in the UI. No messages should be sent back to the websocket.

## Events

Messages from the websocket are always sent as JSON strings which contain the 'event' and 'content' attributes.

| Event            | Content    | Content Type                             |
|------------------|------------|------------------------------------------|
| notice           | Message    | string                                   |
| planCreation     | Plan       | [model.Plan](../model/model.go)          |
| planDeleted      | Plan name  | string                                   | 
| missionCreation  | Mission    | [mission.Mission](../mission/mission.go) |
| missionUpdate    | Mission    | [mission.Mission](../mission/mission.go) |
| missionCompleted | Mission    | [mission.Mission](../mission/mission.go) |
| missionDeleted   | Mission Id | string                                   |

## Authentication

The websocket protocol has no built-in authentication methods. Therefore, Houston requires the API key
to be provided as a query parameter when the connection is first created. 

For example, if connecting from a web client with JavaScript:

```js
let conn = new WebSocket("ws://" + location.host + "/ws?a=" + API_KEY);

conn.onmessage = (message) => {

  const messages = message.data.split('\n');

  messages.forEach(eventJSON => {
    const event = JSON.parse(eventJSON)
    console.log(event.event, event.content)
  })

};
```
