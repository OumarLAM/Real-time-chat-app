const socket = io();

let username;

const setName = () => {
  username = prompt("Wanna join the chat? What's your name ? ");
  socket.emit("set-name", username);
};

socket.on("request-name", () => {
  setName();
});

const form = document.querySelector("form");
const input = document.querySelector("#input");
const messages = document.querySelector("#messages");

form.addEventListener("submit", (e) => {
  e.preventDefault();
  if (!username) {
    setName();
  } else if (input.value) {
    socket.emit("chat-message", input.value);
    input.value = "";
  }
});

socket.on("chat-message", (data) => {
  const item = document.createElement("li");
  if (data.name === username) {
    item.textContent = `Me: ${data.message}`;
    messages.appendChild(item);
  } else {
    item.textContent = `${data.name}: ${data.message}`;
    messages.appendChild(item);
  }
});

socket.on("user-connected", function (message) {
  console.log(message);
});

socket.on("user-disconnected", function (message) {
  console.log(message);
});
