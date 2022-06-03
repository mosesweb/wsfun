import { useEffect, useState } from 'react'
import logo from './logo.svg'
import './App.css'

import * as WebSocket from "websocket"
import React from 'react'
import { ChatList } from './components/chatlist'




function App() {


  const [textElement, setTextElement] = useState<string>("")
  const [socket, setSocket] = useState<WebSocket.w3cwebsocket | null>(null)
  const [messages, setMessage] = useState<string[]>([])

  const sendMsg = () => {
    socket?.send(textElement)
  }
  const onChangeText = (event: any) => {
    
    setTextElement(event.target.value)
  }


  const addMessage = (message: string) => {
    setMessage(m => [...m, message])
  }
  useEffect(() => {
    console.log("in it")
    const socket = new WebSocket.w3cwebsocket('ws://localhost:8080/ws');
    setSocket(socket);
    socket.onopen = function () {
      socket.send("helloheee!")
      socket.onmessage = (data: any) => {
        let dataInJson = JSON.parse(data.data);
        addMessage(dataInJson.message);
      };
    };
  }, []);


  const [count, setCount] = useState(0)

  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>Hello Vite + React!</p>
          <ChatList messages={messages} />
        <p>
          Edit <code>App.tsx</code> and save to test HMR updates.
        </p>
        <input type="text" onChange={onChangeText} className='text-area' />
        <input type="submit" onClick={sendMsg} className='submit' />

      </header>
    </div>
  )
}

export default App
