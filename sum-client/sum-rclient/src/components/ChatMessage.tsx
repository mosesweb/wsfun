import { Component } from "react";

export class ChatMessage extends Component<{ message: string; }> {
    render() {
        return <div>{this.props.message}</div>
    }
}
