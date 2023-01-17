import { ChangeEvent, useEffect, useRef, useState } from 'react'
import './App.css'
import { getAuth, signInWithPopup, FacebookAuthProvider, signOut } from "firebase/auth";
import * as WebSocket from "websocket"
import { initializeApp } from 'firebase/app';

class User {
  displayName: string = "";
  email: string = "";
}
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
  const [clientUser, setclientUser] = useState<User | null>(null)
  const [clientSocket, setClientSocket] = useState<WebSocket.w3cwebsocket | null>(null)

  const [messages, SetMessages] = useState<ChatMessage[]>([]);
  const textarea = useRef<HTMLInputElement>(null);
  const provider = new FacebookAuthProvider();

  const firebaseConfig = {
    apiKey: "AIzaSyCaUQYTcebWzLOj6vVi-6kKkVIKmMPZPmc",
    authDomain: "mychat-359818.firebaseapp.com",
    projectId: "mychat-359818",
    storageBucket: "mychat-359818.appspot.com",
    messagingSenderId: "664404803242",
    appId: "1:664404803242:web:3a6ea67f3674a6f4245ad3",
    measurementId: "G-2WCTT5WJ1E"
  };
  
  const app = initializeApp(firebaseConfig);
  const auth = getAuth();

  const logout = () => {
    console.log("signout")
    auth.signOut().then((e) => 
      {
        setclientUser(null);
        alert("You're logged out.");
      });
  }

  useEffect(() => {
    auth.onAuthStateChanged((user) => {
      console.log(user);

      if (user) {
        const clientUser = new User();
    
        clientUser.displayName = user.displayName ?? "unknown";
        clientUser.email = user.email ?? "unknown";

        console.log(clientUser);
        setclientUser(clientUser);
      }
      else {
       
      }
    })
   
    const socket = new WebSocket.w3cwebsocket('wss://myservice-ggddbbhemq-nw.a.run.app/ws');

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
  
  const send = function(event: any) {
    event.preventDefault();

    console.log("click")
    const msg = JSON.stringify({
      text: messageInput,
      time: +new Date(),
      user: auth.currentUser?.displayName,
    })

    if(messageInput !== "") {
      clientSocket?.send(msg)
    }

    if(textarea.current != null) {
      textarea.current.value = "";
    }
    setMessageInput("");
  }

  const handleInput = (event: ChangeEvent<HTMLInputElement>) => {
    setMessageInput(event.target.value);
  }


  useEffect(() => {
    console.log(messages);
  }, [messages]);

 const login = () => {
  signInWithPopup(auth, provider)
  .then((result: any) => {
    // The signed-in user info.
    const user = result.user; 
    const clientUser = new User();

    clientUser.displayName = user.displayName;
    clientUser.email = user.email;

    console.log(clientUser);
    setclientUser(clientUser);

    // This gives you a Facebook Access Token. You can use it to access the Facebook API.
    const credential = FacebookAuthProvider.credentialFromResult(result);
    if(credential === null) {
      throw("Err cred null");
    }

    const accessToken = credential.accessToken;

    // ...
  })
  .catch((error: any) => {
    // Handle Errors here.
    const errorCode = error.code;
    const errorMessage = error.message;
    // The email of the user's account used.
    const email = error.customData.email;
    // The AuthCredential type that was used.
    const credential = FacebookAuthProvider.credentialFromError(error);

    // ...
  });
 }

  return (
   <> <div className="App">
        <p>Chat</p>
        { auth.currentUser !== null && 
        <>
        <div className="chatbox">
        {
          messages.map((m, i: number) => {
            const thedate = new Date(m.time * 1000);
            return <div className='msg' key={i}>
                <div className="time-info">{thedate.toDateString() + " - " + thedate.toTimeString().substring(0, 8) } <b>{m.user}</b></div>
            {m.text}</div>
          })
        }
        </div>
        <form>
          <p>
          <input type="text" ref={textarea} placeholder="my message" id='msginput' onChange={handleInput} /><br />
          <button type='submit'  className='sendbtn' onClick={send}>SEND</button>
          </p>
        </form>
        </>
        }
        {auth.currentUser === null && 
          <div onClick={() => login()}>Login</div> 
        }
          {auth.currentUser !== null && 
          <div onClick={() => logout()}>Logout {auth.currentUser?.displayName}</div> 
        }
    </div> 
    </>
  )
}

export default App
