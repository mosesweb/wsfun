import { Server } from "socket.io";

const io = new Server({
  // options
});

io.on("connection", (socket) => {
  // ...
});

io.listen(3000);

// server side
io.on('connection', client => { 
    console.log(client.id + "connected!")
 });

