import { io, type Socket } from "socket.io-client";

import { env } from "../../config/env";
import { TIMEOUTS } from "../../config/constants";

type Listener<T> = (payload: T) => void;

/**
 * Thin wrapper around socket.io-client. Reconnects automatically using
 * the supplied auth token. Callers subscribe to typed event channels via
 * `on()` and receive an unsubscribe function.
 */
export class SocketService {
  private socket: Socket | null = null;
  private token: string | null = null;

  connect(token: string): Socket {
    if (this.socket && this.socket.connected && this.token === token) {
      return this.socket;
    }
    this.disconnect();
    this.token = token;
    this.socket = io(env.socketUrl, {
      auth: { token },
      transports: ["websocket"],
      reconnection: true,
      reconnectionDelay: TIMEOUTS.socketReconnectMs,
    });
    return this.socket;
  }

  disconnect(): void {
    if (this.socket) {
      this.socket.removeAllListeners();
      this.socket.disconnect();
      this.socket = null;
    }
    this.token = null;
  }

  isConnected(): boolean {
    return Boolean(this.socket?.connected);
  }

  on<T>(event: string, handler: Listener<T>): () => void {
    if (!this.socket) {
      throw new Error(
        "SocketService: cannot subscribe before connect() is called",
      );
    }
    const socket = this.socket;
    socket.on(event, handler);
    return () => {
      socket.off(event, handler);
    };
  }

  emit<T>(event: string, payload: T): void {
    this.socket?.emit(event, payload);
  }
}

export const socketService = new SocketService();
