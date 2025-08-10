# API v2 Service

A dual-protocol API service that provides both gRPC and HTTP/REST endpoints using grpc-gateway. The service includes basic authentication for protected endpoints and demonstrates various gRPC-Gateway features.

## Architecture

The service runs two servers concurrently:
- **gRPC Server**: Port 10551
- **HTTP Gateway**: Port 10552 (proxies to gRPC server)

## Protobuf Definitions

### Service Definition (`proto/api/v2/service.proto`)

```protobuf
service V2 {
  rpc Hello(HelloRequest) returns (HelloResponse) {
    option (google.api.http) = {
      get : "/v2/hello"
      additional_bindings {get : "/v3/hello"}  // Multi-binding example
    };
  }
  rpc Greetings(GreetRequest) returns (GreetResponse) {
    option (google.api.http) = {
      post : "/v2/greet"
      additional_bindings {get : "/v2/greet/{name}"}  // Path parameter example
    };
  }
  rpc Echo(EchoRequest) returns (EchoResponse) {
    option (google.api.http) = {
      post : "/v2/echo"
      body : "*"  // Request body mapping
    };
  }
}
```

### Message Definitions

**Hello Messages** (`proto/api/v2/hello.proto`):
```protobuf
message HelloRequest {}
message HelloResponse { string msg = 1; }
```

**Greet Messages** (`proto/api/v2/greet.proto`):
```protobuf
message GreetRequest { string name = 1; }
message GreetResponse { string msg = 1; }
```

**Echo Messages** (`proto/api/v2/echo.proto`):
```protobuf
message EchoRequest { map<string, string> data = 1; }
message EchoResponse {
  map<string, string> data = 1;
  map<string, string> headers = 2;
}
```

## Authentication

The service supports basic authentication with the following behavior:

- **No Auth Required**: `/v2/hello`, `/v3/hello`, and `Hello` gRPC method
- **Auth Required**: All other endpoints (`/v2/greet`, `/v2/echo`, and corresponding gRPC methods)

### Authentication Configuration

Configure credentials when creating the service:
```go
api := NewAPIV2(Opts{
    Log: logger,
    Username: "admin",
    Password: "secret",
})
```

If no credentials are provided, authentication is disabled for all endpoints.

## Running the Service

Start the service using:
```bash
make run
```

This will start both the gRPC server on port 10551 and HTTP gateway on port 10552.

## Testing Endpoints

### 1. Hello Endpoint (No Authentication)

Returns a simple greeting message.

**HTTP (both paths work):**
```bash
# Using /v2/hello
curl http://localhost:10552/v2/hello

# Using /v3/hello (additional binding example)
curl http://localhost:10552/v3/hello
```

**Expected Response:**
```json
{"msg":"Hello from API v2!"}
```

**gRPC:**
```bash
grpcurl -plaintext localhost:10551 api.v2.V2/Hello
```

### 2. Greetings Endpoint (Authentication Required)

Demonstrates both POST and GET with path parameters.

**HTTP POST:**
```bash
curl -X POST http://localhost:10552/v2/greet \
  -u admin:secret \
  -H "Content-Type: application/json" \
  -d '{"name":"World"}'
```

**HTTP GET with path parameter:**
```bash
# Greet with name from URL path
curl -u admin:secret http://localhost:10552/v2/greet/Alice

# Greet with default message
curl -u admin:secret http://localhost:10552/v2/greet
```

**Expected Response:**
```json
{"msg":"Hello, World!"}
{"msg":"Hello, Alice!"}
{"msg":"Hello, World!"}
```

**gRPC:**
```bash
grpcurl -plaintext \
  -H "authorization: Basic YWRtaW46c2VjcmV0" \
  -d '{"name":"World"}' \
  localhost:10551 api.v2.V2/Greetings
```

### 3. Echo Endpoint (Authentication Required)

Echoes back the request data and HTTP headers.

**HTTP:**
```bash
curl -X POST http://localhost:10552/v2/echo \
  -u admin:secret \
  -H "Content-Type: application/json" \
  -H "X-Custom-Header: test-value" \
  -d '{"data": {"message":"hello","type":"greeting"}}'
```

**Expected Response:**
```json
{
  "data": {
    "message": "hello",
    "type": "greeting"
  },
  "headers": {
    "authorization": "Basic YWRtaW46c2VjcmV0",
    "content-type": "application/json",
    "x-custom-header": "test-value",
    "user-agent": "curl/7.68.0",
    ...
  }
}
```

**gRPC:**
```bash
grpcurl -plaintext \
  -H "authorization: Basic YWRtaW46c2VjcmV0" \
  -H "x-custom-header: test-value" \
  -d '{"data":{"message":"hello","type":"greeting"}}' \
  localhost:10551 api.v2.V2/Echo
```

## Sample Use Cases

### 1. Multi-Version API Support
The Hello endpoint demonstrates how to support multiple API versions using `additional_bindings`. The same gRPC method serves both `/v2/hello` and `/v3/hello`.

### 2. Path Parameter Extraction
The Greetings endpoint shows how to extract path parameters using `{name}` syntax in the HTTP binding.

### 3. Request/Response Body Mapping
The Echo endpoint demonstrates:
- Full request body mapping with `body: "*"`
- Returning both request data and HTTP headers
- Map-type fields in protobuf messages

### 4. Authentication Patterns
Shows how to implement basic authentication with selective endpoint protection:
- Public endpoints (Hello)
- Protected endpoints (Greetings, Echo)
- Header forwarding from HTTP to gRPC

### 5. Dual Protocol Support
Demonstrates serving the same business logic over both gRPC and HTTP/REST protocols with a single implementation.

## Basic Auth Credentials

For testing, use these default credentials:
- **Username**: `admin`
- **Password**: `secret`
- **Base64 Encoded**: `YWRtaW46c2VjcmV0`

## Error Handling

### Authentication Errors

**HTTP 401 Response:**
```bash
curl http://localhost:10552/v2/greet
# Returns: 401 Unauthorized with WWW-Authenticate header
```

**gRPC Unauthenticated:**
```bash
grpcurl -plaintext localhost:10551 api.v2.V2/Greetings
# Returns: Code: Unauthenticated
```

## Development

The service uses:
- **grpc-gateway/v2** for HTTP/gRPC translation
- **crypto/subtle** for timing-safe authentication
- **google.golang.org/grpc/metadata** for header forwarding
- Standard gRPC interceptors and HTTP middleware patterns
