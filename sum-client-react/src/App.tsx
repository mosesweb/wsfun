import { ChangeEvent, useEffect, useRef, useState } from 'react'
import './App.css'
import { getAuth, signInWithPopup, FacebookAuthProvider, signOut } from "firebase/auth";
import * as WebSocket from "websocket"
import { initializeApp } from 'firebase/app';
import axios from 'axios';

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

class imageResponse {
  texts: textInfo[] = []
  image: string = ""
}
class vertices {
  x: number = 0;
  y: number = 0;
}

class textInfo {
  boundingpoly: vertices[] = [];
  text: string = "";
}
function App() {
  const [messageInput, setMessageInput] = useState("")
  const [clientUser, setclientUser] = useState<User | null>(null)
  const [clientSocket, setClientSocket] = useState<WebSocket.w3cwebsocket | null>(null)

  const [messages, SetMessages] = useState<ChatMessage[]>([]);
  const textarea = useRef<HTMLInputElement>(null);
  const imageFile = useRef<HTMLInputElement>(null);
  const [imageText, SetimageText] = useState<imageResponse>();
  const provider = new FacebookAuthProvider();

  const firebaseConfig = {
    apiKey: process.env.REACT_APP_APIKEY,
    authDomain: process.env.REACT_APP_AUTHDOMAIN,
    projectId: process.env.REACT_APP_PROJECTID,
    storageBucket: process.env.REACT_APP_STORAGEBUCKET,
    messagingSenderId: process.env.REACT_APP_MESSAGESENDERID,
    appId: process.env.REACT_APP_APPID,
    measurementId: process.env.REACT_APP_MEASUREMENTID,
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
   
    // TODO check if localhost is run on https or http then use that env var
    const socket = new WebSocket.w3cwebsocket(process.env.REACT_APP_CHAT_API_WS ?? "");

    socket.onopen = function () {
      setClientSocket(socket);
      socket.onerror = (error: Error) => {
        console.log(error);
        socket.close()
      }

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
    console.log(error);
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

 

 async function uploadFile() {
    console.log(imageFile.current?.files);
    if(!imageFile.current)
      return;
    if(!imageFile.current.files)
      return;
    var formData = new FormData();
    formData.append("image", imageFile.current.files[0]);
    console.log(formData.get("image"));
    axios.post(process.env.REACT_APP_CHAT_API ?? "", 
    formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
    }).then(e => {
      console.log(e.data);
      let dataFixed: string = e.data.replace("(MISSING)","");
      
      const dataResult = JSON.parse(dataFixed) as imageResponse;
      console.log(dataResult);
      SetimageText(dataResult);
    })
  }

  return (
   <> <div className="App">
    <p>Image</p>
    <img src={`data:image/jpeg;charset=utf-8;base64,${imageText?.image}`} />
   
    <div><input type="file" ref={imageFile} name="file"></input>
    <button onClick={() => uploadFile()}>upload</button>
    </div>
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
