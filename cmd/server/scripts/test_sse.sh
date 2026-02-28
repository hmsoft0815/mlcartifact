#!/bin/bash
# Copyright (c) 2026 Michael Lechner. All rights reserved.

PORT=8082
ADDR=":$PORT"
URL="http://localhost:$PORT"

echo "🔨 Building artifact-server..."
make build > /dev/null

echo "🚀 Starting artifact-server in SSE mode on $PORT..."
./build/artifact-server -addr "$ADDR" &
SERVER_PID=$!

trap "kill $SERVER_PID 2>/dev/null" EXIT

sleep 3

echo "📡 Connecting to SSE stream..."
SSE_LOG=$(mktemp)
curl -s -N "$URL/sse" > "$SSE_LOG" &
CURL_PID=$!

sleep 3
ENDPOINT_URL=$(grep "data: " "$SSE_LOG" | head -n 1 | sed 's/data: //' | tr -d '\r')

if [ -z "$ENDPOINT_URL" ]; then
    echo "❌ Failed to get SSE endpoint."
    kill $CURL_PID 2>/dev/null
    exit 1
fi

echo "✅ Connected. Endpoint: $ENDPOINT_URL"

echo "🔑 Initializing..."
curl -s -X POST "$ENDPOINT_URL" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-sse","version":"1"}}}' > /dev/null

sleep 2

echo "🧪 Saving artifact via SSE..."
curl -s -X POST "$ENDPOINT_URL" \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 2,
       "method": "tools/call",
       "params": {
         "name": "write_artifact",
         "arguments": {
           "filename": "sse_test.txt",
           "content": "Stored via SSE Transport"
         }
       }
     }' > /dev/null

echo "📥 Waiting for result in stream..."
sleep 3

echo "--- Received Events ---"
cat "$SSE_LOG"
echo "-----------------------"

if grep -q "sse_test.txt" "$SSE_LOG"; then
    echo "🎉 SUCCESS: Artifact reference found in SSE stream!"
else
    echo "❌ FAILURE: Result not found."
    exit 1
fi

kill $CURL_PID 2>/dev/null
rm "$SSE_LOG"
