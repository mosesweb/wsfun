import React from "react";
import { ChatMessage } from "./ChatMessage";

export class ChatList extends React.Component< ShowModalProps>  {
    date = new Date()
    render(): JSX.Element {
        return (
            <div> 
            {
                this.props.messages.map((m: string) => {
                   return (<ChatMessage message={m} key={m + (+(this.date))} />)
                   })
            }
            </div>
        )
    }
  }

  interface ShowModalProps {
    messages: string[];
}