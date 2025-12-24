#!/bin/bash

# Test ownership authorization
# This script tests that users can only access their own tasks

BASE_URL="http://localhost"

echo "=== Ownership Authorization Test ==="
echo

# 1. Register and login as User A
echo "1. Creating User A..."
USER_A_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test8",
    "password": "test123"
  }')

USER_A_ID=$(echo "$USER_A_RESPONSE" | grep -o '"user_id":[0-9]*' | cut -d':' -f2)
echo "User A ID: $USER_A_ID"

USER_A_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test8",
    "password": "test123"
  }')

USER_A_TOKEN=$(echo "$USER_A_LOGIN" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo "User A Token: ${USER_A_TOKEN:0:50}..."
echo

# 2. Register and login as User B
echo "2. Creating User B..."
USER_B_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser_b",
    "email": "userb@test.com",
    "password": "password123"
  }')

USER_B_ID=$(echo "$USER_B_RESPONSE" | grep -o '"user_id":[0-9]*' | cut -d':' -f2)
echo "User B ID: $USER_B_ID"

USER_B_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "userb@test.com",
    "password": "password123"
  }')

USER_B_TOKEN=$(echo "$USER_B_LOGIN" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo "User B Token: ${USER_B_TOKEN:0:50}..."
echo

# 3. User A creates a task
echo "3. User A creating a task..."
TASK_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER_A_TOKEN" \
  -d '{
    "task_type": "IMAGE_RESIZE"
  }')

TASK_ID=$(echo "$TASK_RESPONSE" | grep -o '"task_id":[0-9]*' | cut -d':' -f2)
echo "Task created with ID: $TASK_ID"
echo

# 4. Test: User A can access their own task by ID (should succeed)
echo "4. TEST: User A accessing their own task (GET /tasks/$TASK_ID)..."
echo "Expected: 200 OK"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET "$BASE_URL/api/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $USER_A_TOKEN")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
  echo "✅ PASS: User A can access their own task (Status: $HTTP_STATUS)"
else
  echo "❌ FAIL: Expected 200, got $HTTP_STATUS"
fi
echo "Response: $BODY"
echo

# 5. Test: User B trying to access User A's task by ID (should fail with 403)
echo "5. TEST: User B accessing User A's task (GET /tasks/$TASK_ID)..."
echo "Expected: 403 Forbidden"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET "$BASE_URL/api/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $USER_B_TOKEN")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "403" ]; then
  echo "✅ PASS: User B cannot access User A's task (Status: $HTTP_STATUS)"
else
  echo "❌ FAIL: Expected 403, got $HTTP_STATUS"
fi
echo "Response: $BODY"
echo

# 6. Test: User A can list their own tasks (should succeed)
echo "6. TEST: User A listing their own tasks (GET /users/$USER_A_ID/tasks)..."
echo "Expected: 200 OK"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET "$BASE_URL/api/v1/users/$USER_A_ID/tasks" \
  -H "Authorization: Bearer $USER_A_TOKEN")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
  echo "✅ PASS: User A can list their own tasks (Status: $HTTP_STATUS)"
else
  echo "❌ FAIL: Expected 200, got $HTTP_STATUS"
fi
echo "Response: $BODY"
echo

# 7. Test: User B trying to list User A's tasks (should fail with 403)
echo "7. TEST: User B listing User A's tasks (GET /users/$USER_A_ID/tasks)..."
echo "Expected: 403 Forbidden"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET "$BASE_URL/api/v1/users/$USER_A_ID/tasks" \
  -H "Authorization: Bearer $USER_B_TOKEN")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "403" ]; then
  echo "✅ PASS: User B cannot list User A's tasks (Status: $HTTP_STATUS)"
else
  echo "❌ FAIL: Expected 403, got $HTTP_STATUS"
fi
echo "Response: $BODY"
echo

# 8. Test: User B can list their own tasks (should succeed, empty list)
echo "8. TEST: User B listing their own tasks (GET /users/$USER_B_ID/tasks)..."
echo "Expected: 200 OK with empty array"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X GET "$BASE_URL/api/v1/users/$USER_B_ID/tasks" \
  -H "Authorization: Bearer $USER_B_TOKEN")
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

if [ "$HTTP_STATUS" = "200" ]; then
  echo "✅ PASS: User B can list their own tasks (Status: $HTTP_STATUS)"
else
  echo "❌ FAIL: Expected 200, got $HTTP_STATUS"
fi
echo "Response: $BODY"
echo

echo "=== Test Summary ==="
echo "All ownership authorization tests completed!"
echo "Expected results:"
echo "  - User A can access their own task by ID: ✅"
echo "  - User B CANNOT access User A's task by ID: ✅"
echo "  - User A can list their own tasks: ✅"
echo "  - User B CANNOT list User A's tasks: ✅"
echo "  - User B can list their own tasks (empty): ✅"
