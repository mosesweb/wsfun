import { ChangeEvent, useEffect, useState } from 'react'
import logo from './logo.svg'
import './App.css'

import * as WebSocket from "websocket"

interface ChatMessage {
  message: string
}


function App() {
  const [count, setCount] = useState(0)
  const [messageInput, setMessageInput] = useState("")
  const [clientSocket, setClientSocket] = useState<WebSocket.w3cwebsocket | null>(null)

  useEffect(() => {
    console.log("in it")
    const socket = new WebSocket.w3cwebsocket('ws://localhost:8080/ws');
    
    socket.onopen = function () {
      setClientSocket(socket);
      socket.send("helloheee!")
      socket.onmessage = (msg: any) => {
        console.log(msg);
        console.log("we got msg..")
      };
    };
  }, []);

  const send = function() {
    console.log("click")
    clientSocket?.send(messageInput)
  }

  const handleInput = function(event: ChangeEvent<HTMLTextAreaElement>) {
    console.log(event.target.value);
    setMessageInput(event.target.value);
  }


  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>Hello Vite + React!</p>
        <p>
         <textarea id='msginput' onChange={handleInput} defaultValue={'Message'}></textarea>
         <button onClick={send} />
        </p>
        <p>
          Edit <code>App.tsx</code> and save to test HMR updates.
        </p>
        <p>
          <a
            className="App-link"
            href="https://reactjs.org"
            target="_blank"
            rel="noopener noreferrer"
          >
            Learn React
          </a>
          {' | '}
          <a
            className="App-link"
            href="https://vitejs.dev/guide/features.html"
            target="_blank"
            rel="noopener noreferrer"
          >
            Vite Docs
          </a>
        </p>
      </header>
    </div>
  )
}

export default App
