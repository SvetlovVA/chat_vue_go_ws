<!-- src/components/Chat.vue -->
<template>
  <div class="chat-container">
    <div class="chat-header">
      <h2>Real-time Chat</h2>
      <div class="connection-status" :class="{ connected: isConnected }">
        {{ isConnected ? 'Connected' : 'Disconnected' }}
      </div>
    </div>

    <div class="chat-messages" ref="messagesContainer">
      <div v-for="(msg, index) in messages" :key="index" class="message">
        <span class="timestamp">{{ formatTime(msg.timestamp) }}</span>
        <strong class="user">{{ msg.userId || msg.user }}:</strong>
        <span class="text">{{ msg.message || msg.text }}</span>
      </div>
    </div>

    <div class="chat-input">
      <input
          v-model="userName"
          placeholder="Your name"
          class="name-input"
          @change="saveUserName"
      />
      <input
          v-model="inputMessage"
          @keyup.enter="sendMessage"
          placeholder="Type your message..."
          :disabled="!isConnected"
          class="message-input"
      />
      <button @click="sendMessage" :disabled="!isConnected || !inputMessage.trim()">
        Send
      </button>
    </div>

    <div class="chat-controls">
      <button @click="toggleConnection" class="connection-btn">
        {{ isConnected ? 'Disconnect' : 'Connect' }}
      </button>
      <button @click="clearMessages" class="clear-btn">
        Clear
      </button>
      <span class="message-count">Messages: {{ messages.length }}</span>
    </div>

    <div class="connection-info" v-if="!isConnected">
      <p>Make sure your Go backend is running on http://0.0.0.0:8080</p>
      <p>Run in terminal: <code>cd ~/projects/vuechat/backend && go run server.go</code></p>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import GrpcClient from '../grpc/client'

export default {
  name: 'Chat',
  setup() {
    const messages = ref([])
    const inputMessage = ref('')
    const userName = ref(localStorage.getItem('chat-username') || `User_${Math.floor(Math.random() * 1000)}`)
    const isConnected = ref(false)
    const messagesContainer = ref(null)

    let client = null
    let unsubscribe = null

    // Сохранение имени пользователя
    const saveUserName = () => {
      localStorage.setItem('chat-username', userName.value)
    }

    // Подключение
    const connect = async () => {
      try {
        client = new GrpcClient('http://localhost:8080')
        await client.connect()

        isConnected.value = true

        // Подписка на сообщения
        unsubscribe = client.onMessage((data) => {
          messages.value.push({
            ...data,
            timestamp: data.timestamp || new Date().toISOString()
          })
          scrollToBottom()
        })

        console.log('Successfully connected to chat server')
      } catch (error) {
        console.error('Failed to connect:', error)
        isConnected.value = false
      }
    }

    // Отключение
    const disconnect = () => {
      if (client) {
        if (unsubscribe) unsubscribe()
        client.disconnect()
        client = null
      }
      isConnected.value = false
    }

    // Переключение подключения
    const toggleConnection = () => {
      if (isConnected.value) {
        disconnect()
      } else {
        connect()
      }
    }

    // Отправка сообщения
    const sendMessage = () => {
      if (!inputMessage.value.trim() || !isConnected.value) return

      try {
        client.sendMessage(userName.value, inputMessage.value.trim())
        inputMessage.value = ''
      } catch (error) {
        console.error('Failed to send message:', error)
        alert('Failed to send message. Please check connection.')
      }
    }

    // Очистка сообщений
    const clearMessages = () => {
      messages.value = []
    }

    // Автоскролл к последнему сообщению
    const scrollToBottom = () => {
      nextTick(() => {
        if (messagesContainer.value) {
          messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
        }
      })
    }

    // Форматирование времени
    const formatTime = (timestamp) => {
      try {
        const date = new Date(timestamp)
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      } catch {
        return '--:--'
      }
    }

    // Автоподключение при монтировании
    onMounted(() => {
      connect()
    })

    // Отключение при размонтировании
    onUnmounted(() => {
      disconnect()
    })

    return {
      messages,
      inputMessage,
      userName,
      isConnected,
      messagesContainer,
      saveUserName,
      sendMessage,
      toggleConnection,
      clearMessages,
      formatTime
    }
  }
}
</script>

<style scoped>
.chat-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
  border: 1px solid #e0e0e0;
  border-radius: 12px;
  background: white;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #4a6fa5;
}

.chat-header h2 {
  margin: 0;
  color: #333;
  font-weight: 600;
}

.connection-status {
  padding: 6px 16px;
  border-radius: 20px;
  font-weight: 600;
  font-size: 0.9em;
  background: #ff6b6b;
  color: white;
  transition: background-color 0.3s ease;
}

.connection-status.connected {
  background: #51cf66;
}

.chat-messages {
  height: 400px;
  overflow-y: auto;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
  margin-bottom: 20px;
}

.message {
  padding: 10px 0;
  border-bottom: 1px solid #e9ecef;
  text-align: left;
}

.message:last-child {
  border-bottom: none;
}

.timestamp {
  color: #6c757d;
  font-size: 0.8em;
  margin-right: 10px;
}

.user {
  color: #4a6fa5;
  margin-right: 8px;
  font-weight: 600;
}

.text {
  color: #495057;
}

.chat-input {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.name-input {
  width: 120px;
  padding: 10px;
  border: 1px solid #ced4da;
  border-radius: 6px;
  font-size: 14px;
}

.message-input {
  flex: 1;
  padding: 10px;
  border: 1px solid #ced4da;
  border-radius: 6px;
  font-size: 16px;
}

.message-input:disabled {
  background: #e9ecef;
  cursor: not-allowed;
}

button {
  padding: 10px 20px;
  background: #4a6fa5;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  transition: background-color 0.3s ease;
}

button:hover:not(:disabled) {
  background: #3a5a8a;
}

button:disabled {
  background: #adb5bd;
  cursor: not-allowed;
}

.chat-controls {
  display: flex;
  gap: 12px;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #e9ecef;
}

.connection-btn {
  background: #51cf66;
}

.connection-btn:hover:not(:disabled) {
  background: #40c057;
}

.clear-btn {
  background: #ff6b6b;
}

.clear-btn:hover:not(:disabled) {
  background: #fa5252;
}

.message-count {
  margin-left: auto;
  color: #6c757d;
  font-size: 0.9em;
}

.connection-info {
  margin-top: 20px;
  padding: 15px;
  background: #fff3cd;
  border: 1px solid #ffeaa7;
  border-radius: 6px;
  color: #856404;
  font-size: 0.9em;
  text-align: left;
}

.connection-info p {
  margin: 5px 0;
}

.connection-info code {
  background: #f1f3f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}
</style>