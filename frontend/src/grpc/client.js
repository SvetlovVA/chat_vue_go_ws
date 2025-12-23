// src/grpc/client.js
export default class ChatClient {
    constructor(host = 'http://localhost:8080') {
        this.host = host;
        this.ws = null;
        this.messageListeners = [];
    }

    // Подключение к WebSocket
    connect() {
        return new Promise((resolve, reject) => {
            const wsUrl = this.host.replace('http://', 'ws://') + '/ws';
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected successfully');
                resolve();
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.messageListeners.forEach(listener => listener(data));
                } catch (error) {
                    console.error('Error parsing message:', error);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                reject(error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.ws = null;
            };
        });
    }

    // Отправка сообщения
    sendMessage(userId, message, room = 'general') {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            throw new Error('WebSocket not connected');
        }

        const payload = {
            userId,
            message,
            room,
            timestamp: new Date().toISOString()
        };

        this.ws.send(JSON.stringify(payload));
        return payload;
    }

    onMessage(callback) {
        this.messageListeners.push(callback);
        return () => {
            const index = this.messageListeners.indexOf(callback);
            if (index > -1) this.messageListeners.splice(index, 1);
        };
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.messageListeners = [];
    }

    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}