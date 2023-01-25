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
  const text = `"AEON Market\nピーコックストア代官山店\nTEL03-6415-3051 FAX03-6415-3055\n領収証\nイオンマーケット株式会社\n2023/ 1/8(日)\n11:39\nレジ0132\n取3763 132セルフレジ\nポッカレモン100\n小計\n外税 8%対象額\n外税8%\n合計\nクレジット\nお釣り\n本人確認省略\nカード会社\n会員番号\nお取扱日\n取引内容\n伝票番号\n取扱区分\n金\n額\n承認番号\nAID\nAPL\n458X\nお買上商品数: 1\n※印は軽減税率8%対象商品\n[クレジットカード売上票]\n( お客様控え )\n¥458\n¥458\n¥36\n¥494\n¥494\n¥0\nお買上\nJCB 35730\nXXXX-XXXX-XXXX-2982\n2023年 01月08日\n(IC)\n132516\n10\n#494\n0454099\nA0000000651010\nJCB Debit\nWAON POINT 会員募集中!\n今すぐ会員登録でオトクにお買物!\nhttp://www.smartwaon.com\n[スマートワオン] で検索","AEON","Market","ピーコック","ストア","代官山","店","TEL03-6415-3051","FAX03-6415-3055","領","収","証","イオン","マーケット","株式会社","2023","/","1/8","(","日",")","11:39","レジ","0132","取","3763","132","セルフレジ","ポッカ","レモン","100","小","計","外","税","8","%","対象","額","外","税","8","%","合計","クレジット","お","釣り","本人","確認","省略","カード","会社","会員","番号","お","取扱","日","取","引","内容","伝","票","番号","取","扱","区分","金","額","承認","番号","AID","APL","458X","お","買上","商品","数",":","1","※","印","は","軽減","税率","8","%","対象","商品","[","クレジット","カード","売上","票","]","(","お客様","控え",")","¥","458","¥","458","¥","36","¥","494","¥","494","¥","0","お","買上","JCB","35730","XXXX","-","XXXX","-","XXXX","-","2982","2023","年","01","月","08","日","(","IC",")","132516","10","#","494","0454099","A0000000651010","JCB","Debit","WAON","POINT","会員","募集","中","!","今","すぐ","会員","登録","で","オトク","に","お","買物","!","http://www.smartwaon.com","[","スマート","ワオン","]","で","検索",`
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
      console.log(e);
      const dataResult = (e.data as imageResponse);
      console.log(dataResult);
      SetimageText(dataResult);
    })
  }

  return (
   <> <div className="App">
    <p>Image</p>
    <img src={`data:image/jpeg;charset=utf-8;base64,${imageText?.image}`} />
    <div>{imageText?.texts.length} - {imageText?.texts.map((t, i) => {
      return <div key={i}>{t.text} {JSON.stringify(t.boundingpoly)}</div>
    })}</div>
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
