var exampleSocket = new WebSocket("ws://localhost:8080");

// Send message every five seconds
setInterval(function(){
  exampleSocket.send('{"recipient": "dick", "message": "sup\'"}')
}, 5000);

// Log received messages
exampleSocket.onmessage = function (event) {
  console.log(event.data);
}
