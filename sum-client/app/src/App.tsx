import { ChangeEvent, useCallback, useEffect, useRef, useState } from 'react'
import logo from './logo.svg'
import './App.css'

import * as WebSocket from "websocket"

class ChatMessage {
  time: number = 0
  text: string = "";
  user: string = "";

  static NewMessage(text: string, time: number, user: string) {
    const newMessage = new ChatMessage();
    newMessage.text = text;
    newMessage.time = time;
    newMessage.user = user;
    return newMessage;
  }
}


function App() {
  const [messageInput, setMessageInput] = useState("")
  const [userInput, setuserInput] = useState(localStorage.getItem("username"))
  const [clientSocket, setClientSocket] = useState<WebSocket.w3cwebsocket | null>(null)

  const [messages, SetMessages] = useState<ChatMessage[]>([]);
  const textarea = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    console.log("in it")
    const socket = new WebSocket.w3cwebsocket('ws://192.168.1.221:3080/ws');

    socket.onopen = function () {
      setClientSocket(socket);
      socket.onmessage = (msg: any) => {

        var obj: ChatMessage[] = JSON.parse(msg.data);
        if(!obj) {
          obj = [];
        }
        for(var i = 0; i < obj.length; i++) {
          if(obj[i] === undefined) {
            console.log("undefined")
            continue;
          }
          console.log(obj[0].text);

        if(messages.findIndex(m => m.time !== obj[i].time)) {
          console.log("why", obj[i]);
          const object = obj[i];
          SetMessages(messagehistory => [...messagehistory,
          ChatMessage.NewMessage(object.text, object.time, object.user)].sort((a, b) => b.time - a.time))
        }
      }
      };
    };

  }, []);


  const send = function() {
    console.log("click")
    const msg = JSON.stringify({
      text: messageInput,
      time: +new Date(),
      user: userInput,
    })

    if(messageInput != "")
    clientSocket?.send(msg)

    if(textarea.current != null) {
      textarea.current.value = "";
    }
    setMessageInput("");
  }

  const handleInput = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setMessageInput(event.target.value);
  }

  const handleuserInput = (event: ChangeEvent<HTMLInputElement>)  => {
    localStorage.setItem("username", event.target.value);
    setuserInput(event.target.value);
  }

  useEffect(() => {
    console.log(messages);
  }, [messages]);


  return (
    <div className="App">
        <p>Chat</p>
        <div className="chatbox">
        {
          messages.map((m, i: number) => {
            return <div className='msg' key={i}>{m.user}: {m.text}</div>
          })
        }
        </div>
        <p>
         <textarea ref={textarea} placeholder="my message" id='msginput' onChange={handleInput}></textarea><br />
         <button  className='sendbtn' onClick={send}>SEND</button>
        </p>
        <p>
          <input type="text" placeholder={userInput ?? ""} onChange={handleuserInput} />
        </p>
    </div>
  )
}

export default App
