var username = window.location.hash.substr(1);

var exampleSocket = new WebSocket("ws://localhost:4691/?user=" + username);

messageList = document.querySelector("#messages > ul");
recipient = document.getElementById("recipient");
msgBox = document.getElementById("msg");

// Send messages when needed
document.getElementById("chatbox").onsubmit = function() {
  exampleSocket.send(JSON.stringify({
    "sender": username,
    "recipient": recipient.value,
    "message": msgBox.value
  }));
  addMessage(username, msgBox.value)
  msgBox.value = "";
  return false;
}

function addMessage(user, value) {
  var msgLi = document.createElement("li");
  msgLi.innerHTML = "<strong>" + user + "</strong> (" + (new Date()).toLocaleTimeString() + "): " + value;
  messageList.appendChild(msgLi);
}

// Log received messages
exampleSocket.onmessage = function (event) {
  console.log(event.data);

  msg = JSON.parse(event.data);
  // Let's only print message from "real" humans, not from our backend
  if ("sender" in msg && "recipient" in msg) {
    addMessage(msg.sender, msg.message);
  }
}
