// src/api/wsClient.ts
import { getConfig } from '../config';

export interface OperationMessage {
    opId: string;
    opType: string; // 'copy' | 'move' | 'delete' | 'rename'
    opName?: string | null; // e.g. "Copying file name to destination"
    opStatus: string; // 'queued' | 'starting' | 'in-progress' | 'completed' | 'error'
    opPercentage?: number | null;
    opSpeed?: string | null;
    opFileCount?: string | null;
    error?: string | null;
    destDir?: string | null;
}

type WsCallback = (data: OperationMessage) => void;

class WebSocketClient {
    private socket: WebSocket | null = null;
    private reconnectTimeout = 5000;
    private listeners: WsCallback[] = [];
    private statusListeners: ((connected: boolean) => void)[] = [];
    private isReconnecting = false;
    private _isConnected = false;
    private pingInterval: ReturnType<typeof setInterval> | null = null;

    private get url() {
        const config = getConfig();
        let baseWs = config.wsUrl;
        
        // Remove trailing slash if it exists
        baseWs = baseWs.replace(/\/$/, '');

        const hasUserPath = baseWs.endsWith('/api/user/ws');
        const hasSharePath = baseWs.endsWith('/api/share/file/ws');
        
        if (window.location.pathname.startsWith('/share')) {
            const pathParts = window.location.pathname.split('/');
            const shareId = pathParts[1] === 'share' ? pathParts[2] : null;
            const qs = shareId ? `?share_id=${shareId}` : '';
            
            if (hasUserPath) {
                return baseWs.replace('/api/user/ws', `/api/share/file/ws${qs}`);
            } else if (hasSharePath) {
                return `${baseWs}${qs}`;
            } else {
                // If it's just a base URL (like wss://drive.leonkhoo.com)
                return `${baseWs}/api/share/file/ws${qs}`;
            }
        }
        
        // Normal user websocket connection
        if (!hasUserPath && !hasSharePath) {
            // If the user provided a base URL in config (like wss://drive.leonkhoo.com),
            // we dynamically append the normal user websocket path.
            return `${baseWs}/api/user/ws`;
        }
        
        return baseWs;
    }

    public subscribeStatus(callback: (connected: boolean) => void) {
        this.statusListeners.push(callback);
        callback(this._isConnected);
        return () => {
            this.statusListeners = this.statusListeners.filter(cb => cb !== callback);
        };
    }

    private updateStatus(connected: boolean) {
        if (this._isConnected !== connected) {
            this._isConnected = connected;
            this.statusListeners.forEach(cb => { cb(connected); });
        }
    }

    public subscribe(callback: WsCallback) {
        this.listeners.push(callback);
        return () => {
            this.listeners = this.listeners.filter(cb => cb !== callback);
        };
    }

    public connect() {
        if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
            return;
        }

        console.log(`[WebSocket] Connecting to ${this.url}...`);
        this.socket = new WebSocket(this.url);

        this.socket.onopen = () => {
            console.log('[WebSocket] Connected');
            this.updateStatus(true);
            
            // Start a Heartbeat (Ping) to prevent idle timeouts
            this.pingInterval = setInterval(() => {
                if (this.socket?.readyState === WebSocket.OPEN) {
                    this.socket.send(JSON.stringify({ type: 'ping', opId: 'ping' }));
                }
            }, 30000); // 30 seconds

            if (this.isReconnecting) {
                // Trigger a custom event or let the status listener handle it.
                // We'll dispatch a custom event on window for the successful reconnect.
                window.dispatchEvent(new CustomEvent('ws-reconnected'));
                this.isReconnecting = false;
            }
        };

        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(String(event.data)) as OperationMessage;
                console.log('[WebSocket] Received:', data);
                this.listeners.forEach(cb => { cb(data); });
            } catch {
                console.log('[WebSocket] Received (raw):', event.data);
            }
        };

        this.socket.onclose = () => {
            console.log(`[WebSocket] Disconnected. Reconnecting in ${this.reconnectTimeout / 1000}s...`);
            this.updateStatus(false);
            this.isReconnecting = true;
            
            if (this.pingInterval) {
                clearInterval(this.pingInterval);
                this.pingInterval = null;
            }
            
            setTimeout(() => { this.connect(); }, this.reconnectTimeout);
        };

        this.socket.onerror = (error) => {
            console.error('[WebSocket] Error:', error);
            this.socket?.close();
        };
    }

    public send(data: any) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify(data));
        } else {
            console.error('[WebSocket] Cannot send message, socket not connected');
        }
    }
}

export const wsClient = new WebSocketClient();
