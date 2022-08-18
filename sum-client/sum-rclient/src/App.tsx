import { ChangeEvent, useEffect, useRef, useState } from 'react'
import logo from './logo.svg'
import './App.css'

import * as WebSocket from "websocket"

class ChatMessage {
  time: number = 0
  text: string = "";

  static NewMessage(text: string, time: number) {
    const newMessage = new ChatMessage();
    newMessage.text = text;
    newMessage.time = time;

    return newMessage;
  }
}


function App() {
  const [count, setCount] = useState(0)
  const [messageInput, setMessageInput] = useState("")
  const [usernameInput, setUsernameInput] = useState("an user")
  const [clientSocket, setClientSocket] = useState<WebSocket.w3cwebsocket | null>(null)

  const [messages, SetMessages] = useState<ChatMessage[]>([]);
  const textarea = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    console.log("in it")
    const socket = new WebSocket.w3cwebsocket('ws://localhost:8080/ws');

    socket.onopen = function () {
      setClientSocket(socket);
      socket.send("helloheee!")
      socket.onmessage = (msg: any) => {
        console.log("we got msg..", msg);
        var obj = JSON.parse(msg.data);
        if(messages.findIndex(m => m.time !== obj.time)) {
          SetMessages(messagehistory => [...messagehistory,
            ChatMessage.NewMessage(obj.text, obj.time)].sort((a, b) => b.time - a.time))
        }
      };
    };
  }, []);

  const send = function() {
    console.log("click")
    const msg = JSON.stringify({
      text: messageInput,
      time: +new Date(),
      user: usernameInput,
    })
    clientSocket?.send(msg)
    if(textarea.current != null)
      textarea.current.value = "";
  }

  const handleInput = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setMessageInput(event.target.value);
  }

  const handleUsernameInput = (event: ChangeEvent<HTMLInputElement>)  => {
    setUsernameInput(event.target.value);
  }

  useEffect(() => {
    console.log(messages);
  }, [messages]);


  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>Fun chat {messages.length}</p>
        <div className="chatbox">
        {
          messages.map((m, i: number) => {
            return <div key={i}>{m.text}</div>
          })
        }
        </div>
        <p>
         <textarea ref={textarea} placeholder="my message" id='msginput' onChange={handleInput}></textarea><br />
         <button  className='sendbtn' onClick={send}>SEND</button>
        </p>
        <p>
          <input type="text" placeholder={"an user"} onChange={handleUsernameInput} />
        </p>
      </header>
    </div>
  )
}

export default App
