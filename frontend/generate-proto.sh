#!/bin/bash

echo "=== Генерация proto файлов ==="

# Очистите старые файлы
rm -rf src/generated/*
mkdir -p src/generated

# Генерация JavaScript файлов для сообщений
echo "1. Генерация сообщений..."
protoc -I=../backend/proto \
  --js_out=import_style=commonjs,binary:src/generated \
  ../backend/proto/chat.proto

# Генерация gRPC-web файлов для сервиса
echo "2. Генерация gRPC-web сервиса..."
protoc -I=../backend/proto \
  --grpc-web_out=import_style=commonjs+dts,mode=grpcwebtext:src/generated \
  --plugin=protoc-gen-grpc-web=./bin/protoc-gen-grpc-web \
  ../backend/proto/chat.proto

# Создайте index файл для удобного импорта
echo "3. Создание index файлов..."
cat > src/generated/index.js << 'INDEX_EOF'
// Экспорт всех сгенерированных модулей
module.exports = {
  ...require('./chat_pb'),
  ...require('./chat_pb_service')
}
INDEX_EOF

cat > src/generated/index.d.ts << 'TYPES_EOF'
// TypeScript определения
export * from './chat_pb'
export * from './chat_pb_service'
TYPES_EOF

echo "4. Проверка сгенерированных файлов..."
ls -la src/generated/

echo "=== Генерация завершена ==="
