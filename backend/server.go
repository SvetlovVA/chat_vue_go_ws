package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "backend/gen"
	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

// Добавьте после импортов
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешаем все origins для разработки
	},
}

// Добавьте этот метод в chatServer
func (s *chatServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket client connected")

	// Регистрируем клиента
	s.mu.Lock()
	wsClient := &wsClient{
		conn: conn,
		room: "general",
	}
	s.wsClients = append(s.wsClients, wsClient)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		// Удаляем клиента
		for i, c := range s.wsClients {
			if c == wsClient {
				s.wsClients = append(s.wsClients[:i], s.wsClients[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	// Чтение сообщений от клиента
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("WebSocket message: %v", msg)

		// Рассылаем сообщение всем WebSocket клиентам
		s.mu.RLock()
		for _, client := range s.wsClients {
			if client.room == wsClient.room {
				client.conn.WriteJSON(map[string]interface{}{
					"userId":    msg["userId"],
					"message":   msg["message"],
					"room":      msg["room"],
					"timestamp": time.Now().Format(time.RFC3339),
				})
			}
		}
		s.mu.RUnlock()
	}
}

// Добавьте эту структуру
type wsClient struct {
	conn *websocket.Conn
	room string
}

// Добавьте это поле в chatServer
type chatServer struct {
	pb.UnimplementedChatServiceServer
	mu        sync.RWMutex
	rooms     map[string][]*streamClient
	wsClients []*wsClient // Добавьте это
}

type streamClient struct {
	stream pb.ChatService_ChatStreamServer
	userID string
	room   string
}

func NewChatServer() *chatServer {
	return &chatServer{
		rooms: make(map[string][]*streamClient),
	}
}

// Unary RPC - отправка сообщения
func (s *chatServer) SendMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
	log.Printf("Message from %s: %s", req.UserId, req.Message)

	// Рассылка сообщения всем подписчикам комнаты
	s.broadcastMessage(req.Room, &pb.MessageResponse{
		UserId:    req.UserId,
		Message:   req.Message,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	return &pb.MessageResponse{
		UserId:    req.UserId,
		Message:   "Message received",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// Server Streaming - подписка на сообщения
func (s *chatServer) SubscribeMessages(req *pb.SubscribeRequest, stream pb.ChatService_SubscribeMessagesServer) error {
	// Создаем канал для сообщений
	msgChan := make(chan *pb.MessageResponse)

	// Регистрируем клиента
	s.registerClient(req.Room, req.UserId, msgChan)
	defer s.unregisterClient(req.Room, req.UserId, msgChan)

	// Heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-msgChan:
			if err := stream.Send(msg); err != nil {
				return err
			}
		case <-ticker.C:
			// Отправляем heartbeat
			if err := stream.Send(&pb.MessageResponse{
				UserId:    "server",
				Message:   "ping",
				Timestamp: time.Now().Format(time.RFC3339),
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// Bidirectional Streaming
func (s *chatServer) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	// Регистрируем клиента для bidirectional streaming
	client := &streamClient{
		stream: stream,
		userID: "anonymous", // в реальном приложении получаем из auth
		room:   "general",
	}

	s.mu.Lock()
	s.rooms[client.room] = append(s.rooms[client.room], client)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		// Удаляем клиента из комнаты
		for i, c := range s.rooms[client.room] {
			if c == client {
				s.rooms[client.room] = append(s.rooms[client.room][:i], s.rooms[client.room][i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	// Горутина для отправки сообщений клиенту
	errChan := make(chan error)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}

			log.Printf("Received from %s: %s", req.UserId, req.Message)

			// Обновляем информацию о клиенте
			client.userID = req.UserId
			client.room = req.Room

			// Рассылаем сообщение всем в комнате
			s.broadcastMessage(req.Room, &pb.MessageResponse{
				UserId:    req.UserId,
				Message:   req.Message,
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}
	}()

	// Heartbeat loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			return err
		case <-ticker.C:
			// Отправляем heartbeat
			if err := stream.Send(&pb.MessageResponse{
				UserId:    "server",
				Message:   "heartbeat",
				Timestamp: time.Now().Format(time.RFC3339),
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (s *chatServer) broadcastMessage(room string, msg *pb.MessageResponse) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.rooms[room] {
		go func(c *streamClient) {
			if err := c.stream.Send(msg); err != nil {
				log.Printf("Failed to send to client %s: %v", c.userID, err)
			}
		}(client)
	}
}

// Вспомогательные методы для server streaming
func (s *chatServer) registerClient(room, userID string, msgChan chan<- *pb.MessageResponse) {
	// В реальном приложении здесь была бы мапа каналов
}

func (s *chatServer) unregisterClient(room, userID string, msgChan chan<- *pb.MessageResponse) {
	close(msgChan)
}

func main() {
	// Создаем gRPC сервер
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	chatServer := NewChatServer()
	pb.RegisterChatServiceServer(grpcServer, chatServer)

	// Добавляем WebSocket endpoint
	http.HandleFunc("/ws", chatServer.handleWebSocket)

	// Добавляем простые HTTP endpoints для тестирования
	http.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var msg struct {
				UserId  string `json:"userId"`
				Message string `json:"message"`
				Room    string `json:"room"`
			}

			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":   true,
				"message":   "Message received",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
	})

	// Запускаем gRPC сервер
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	// Обертка для gRPC-Web
	wrappedServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true // В production настройте CORS правильно
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
	)

	// HTTP сервер для gRPC-Web
	httpServer := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wrappedServer.IsGrpcWebRequest(r) || wrappedServer.IsAcceptableGrpcCorsRequest(r) {
				wrappedServer.ServeHTTP(w, r)
			} else {
				// Статические файлы или другие обработчики
				http.DefaultServeMux.ServeHTTP(w, r)
			}
		}),
	}

	// Запускаем оба сервера
	go func() {
		log.Printf("gRPC server listening on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	log.Printf("gRPC-Web server listening on :8080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// Интерсепторы
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("Unary call: %s", info.FullMethod)
	return handler(ctx, req)
}

func streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Printf("Stream call: %s", info.FullMethod)
	return handler(srv, stream)
}
