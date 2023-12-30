const socket = new WebSocket('ws://localhost:8080/ws');

socket.onopen = (event) => {
  console.log("Websocket connection opened:", event);
}

socket.onmessage = (event) => {
  const messages = document.getElementById('messages');
  const li = document.createElement('li');
  li.appendChild(document.createTextNode(event.data));
  messages.appendChild(li);
}

document.getElementById("form").addEventListener("submit", function (event) {
  event.preventDefault();
  const input = document.getElementById("input");
  const message = input.value.trim();
  if (message !== "") {
    socket.send(message);
    input.value = "";
  }
});