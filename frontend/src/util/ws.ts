import { log } from './errors';

export class WS {
    private socket: WebSocket;

    constructor(
        address: string,
        onMessage?: (event: MessageEvent) => void,
        onError?: (event: Event) => void,
        onOpen?: (event: Event) => void,
        onClose?: (event: Event) => void
    ) {
        this.socket = new WebSocket(address);
        this.socket.addEventListener('open', onOpen ?? this._onOpen);
        this.socket.addEventListener('message', onMessage ?? this._onMessage);
        this.socket.addEventListener('error', onError ?? this._onError);
        this.socket.addEventListener('close', onClose ?? this._onClose);
    }

    _onOpen() {
        log('ws opened');
    }

    _onMessage(event: MessageEvent) {
        log('ws msg', event.data);
    }

    _onError(event: Event) {
        log(`ws error ${event}`);
    }

    _onClose() {
        log('ws closed');
    }
}
