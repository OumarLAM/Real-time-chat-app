const express = require("express");
const http = require("http");
const socketIO = require("socket.io");

const app = express();
const server = http.createServer(app);
const io = socketIO(server);

app.use(express.static("public"));

const users = {};

const PORT = process.env.PORT || 3000;

io.on("connection", (socket) => {
  if (!users[socket.id]) {
    socket.emit("request-name");
  }

  socket.on("set-name", (name) => {
    users[socket.id] = name;
    console.log(`A new user named ${name} joined the server`);
  });

  socket.on("disconnect", () => {
    console.log(`${users[socket.id]} unfortunately left the chat!`);
    delete users[socket.id];
  });

  socket.on("chat-message", (message) => {
    io.emit("chat-message", {name: users[socket.id], message });
  });
});

server.listen(PORT, () => {
  console.log(`Server is running on http://localhost:${PORT}`);
});
