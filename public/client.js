const socket = new WebSocket("ws://localhost:8000/ws");

window.onload = function () {
  var form = document.getElementById("messageForm");
  var input = document.getElementById("messageInput");
  var messageDisplay = document.getElementById("messageDisplay");

  form.addEventListener("submit", function (e) {
    e.preventDefault();
    if (input.value) {
      sendMsg(input.value);
      input.value = "";
    }
  });

  connect(function (msg) {
    var item = document.createElement("div");
    var messageData = JSON.parse(msg.data);
    item.textContent = messageData.message;
    messageDisplay.appendChild(item);
    scrollTo(0, document.body.scrollHeight);
  });
};

let connect = (cb) => {
  console.log("Attempting connection...");

  socket.onopen = () => {
    console.log("Successfully connected!!!");
  };

  socket.onmessage = (msg) => {
    console.log(msg);
    cb(msg);
  };

  socket.onclose = (event) => {
    console.log("Socket Closed Connection", event);
  };

  socket.onerror = (error) => {
    console.log("Socket Error", error);
  };
};

let sendMsg = (msg) => {
  console.log("Sending Message: ", msg);
  socket.send(JSON.stringify({Message: msg}));
};
