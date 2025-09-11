# Go RBAC & Authentication API

A secure, production-ready Go API with Paseto authentication, role-based access control (RBAC), and consistent error handling using Google's RPC status codes.

-----

## ⚙️ Tech Stack

* **Language:** Go (Golang)
* **Web Framework:** [Echo v4](https://echo.labstack.com/)
* **Database:** PostgreSQL
* **SQL Builder:** [Squirrel](https://github.com/Masterminds/squirrel) - Fluent SQL generator for Go
* **Database Migrations:** [Migrate](https://github.com/golang-migrate/migrate) - CLI and Go library for database migrations
* **Caching:** [go-redis](https://github.com/go-redis/redis) with Redis
* **Authentication:** [Paseto](https://paseto.io/) - Platform-Agnostic Security Tokens
* **RPC & Error Handling:** [Buf](https://buf.build/) for generating RPC status codes from Google's `status.proto`
* **Logging & Tracing:** **[zap](https://github.com/uber-go/zap)** for structured logging and [OpenTelemetry](https://opentelemetry.io/) for application performance monitoring
* **Containerization:** Docker and Docker Compose

-----

## 📂 Project Structure

This layout keeps all non-reusable code, including services and utilities, within the `internal` directory.

```bash
├── cmd/                     # Main application entry points
│   └── main.go              # Application entry point, handles config and dependency injection
├── migrations/              # Database migration files (.sql)
├── proto/                   # Google RPC status proto definitions
│   └── http/v1/status.proto
│   └── buf.yaml             # Buf linting and schema configuration
├── genproto/                # Generated Go files from Buf
├── internal/                # Private application code
│   ├── auth/                # Paseto and password hashing logic
│   │   ├── service.go
│   │   └── sql.go
│   ├── server/              # HTTP handlers and Echo router setup
│   │   ├── server.go
│   ├── users/               # User-related business logic and repository
│   │   ├── service.go
│   │   └── sql.go
│   ├── roles/               # Role-related business logic and repository
│   │   ├── service.go
│   │   └── sql.go
│   ├── cache/               # Redis caching service
│   │   └── cache.go
├── .env.example
├── .gitignore
├── go.mod                   # Go module dependencies
├── go.sum
├── buf.gen.yaml             # Buf generation configuration
└── README.md
```

-----

## 🔑 Key Architectural Decisions

* **Centralized Configuration:** All application configuration and database connections are handled within `cmd/main.go`. This keeps the setup centralized and easy to manage.
* **Dependency Injection:** Database and service dependencies are injected into each package's **Service struct**. This promotes loose coupling and makes the code easier to test.
* **Encapsulated Logic:** Within each service package (e.g., `internal/users`), a `service.go` file defines the public-facing methods, while a **private `sql.go` file** contains the specific database query logic. This ensures that a service can only be accessed through its exposed public interface.
* **Service-to-Service Communication:** Services that need to interact with each other will do so by holding a reference to the other service's **public Service struct**, ensuring they only use approved, public methods. This prevents direct access to internal database queries from other services.

-----

## 🚀 Getting Started

### Prerequisites

* Go (v1.25+)
* Docker and Docker Compose
* PostgreSQL
* Redis
* **Buf CLI**: For generating gRPC/API-related code.

### Installation

1. **Clone the repository**

    ```bash
    git clone <repository-url>
    cd <project-name>
    ```

2. **Set up environment variables**

    ```bash
    cp .env.example .env
    # Edit .env with your configuration
    ```

3. **Install dependencies**

    ```bash
    go mod tidy
    ```

4. **Run Buf to generate RPC/HTTP code**

    ```bash
    buf generate 
    ```

### Running the Application

* **Using Docker**

    ```bash
    # Build and start containers
    docker-compose up --build -d

    # Run migrations
    docker-compose exec app migrate -path=/migrations -database "postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$POSTGRES_DB?sslmode=disable" up

    # View logs
    docker-compose logs -f app

    # Stop containers
    docker-compose down
    ```

-----

## 📋 API Specification

* **RESTful Architecture:** The API follows RESTful principles, using standard HTTP methods and clear, resource-based URLs.
* **Authentication & Authorization:** Uses **Paseto** for tokens and a custom **RBAC system** for permissions.
* **Error Handling:** Utilizes **Google RPC Status Codes** for consistent, machine-readable error responses.
* **SQL Builder:** All SQL queries are built using **Squirrel** to prevent SQL injection.
* **Data Serialization:** JSON is used for all request and response bodies.

## 🔗 API Endpoints

* **Authentication**

  * `POST /api/v1/auth/register` - Registers a new user.
  * `POST /api/v1/auth/login` - Authenticates a user and returns a Paseto token.
  * `POST /api/v1/auth/refresh` - Refreshes an expired Paseto token.
  * `POST /api/v1/auth/logout` - Logs out the current user.
  * `GET /api/v1/auth/profile` - Retrieves the current user's profile. **(Requires authentication)**

* **Users**

  * `GET /api/v1/users` - Lists all users. **(Requires `users:read` permission)**
  * `GET /api/v1/users/{id}` - Retrieves a user by ID. **(Requires `users:read` permission)**
  * `POST /api/v1/users` - Creates a new user. **(Requires `users:create` permission)**
  * `PUT /api/v1/users/{id}` - Updates a user. **(Requires `users:update` permission)**
  * `DELETE /api/v1/users/{id}` - Deletes a user. **(Requires `users:delete` permission)**

* **Roles**

  * `GET /api/v1/roles` - Lists all roles. **(Requires `roles:read` permission)**
  * `GET /api/v1/roles/{id}` - Retrieves a role by ID. **(Requires `roles:read` permission)**
  * `POST /api/v1/roles` - Creates a new role. **(Requires `roles:create` permission)**
  * `PUT /api/v1/roles/{id}` - Updates a role. **(Requires `roles:update` permission)**
  * `DELETE /api/v1/roles/{id}` - Deletes a role. **(Requires `roles:delete` permission)**

-----

## 🚨 Error Responses

Your API uses Google RPC Status codes to provide consistent and machine-readable error responses. All errors are returned as JSON objects with a standard structure.

### Error Object Structure

```json
{
  "error": {
    "code": 400,
    "message": "Credentials are not valid or incomplete",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.BadRequest",
        "fieldViolations": [
          {
            "field": "email",
            "description": "Email must be a valid email address"
          },
          {
            "field": "password",
            "description": "Password must not be empty"
          }
        ]
      }
    ]
  }
}
```

* `code`: The HTTP status code (e.g., 401).
* `status`: The canonical Google RPC status string (e.g., `UNAUTHENTICATED`).
* `message`: A developer-friendly error message.
* `details`: An optional array of structured objects providing more context about the error.

### Example Error Cases

* **Unauthorized Access (`401 UNAUTHENTICATED`)**
    This error occurs when a request is made to a protected endpoint without a valid Paseto token.

    ```json
    {
      "error": {
        "code": 401,
        "message": "Your provided token is not valid. Please provide a valid token",
        "status": "UNAUTHENTICATED"
      }
    }
    ```

* **Forbidden Access (`403 PERMISSION_DENIED`)**
    This error is returned when a user is authenticated but does not have the necessary permissions to access a resource.

    ```json
    {
      "error": {
          "code": 403,
          "message": "You do not have the 'users:create' permission.",
          "status": "PERMISSION_DENIED"
      }
    }
    ```

* **Bad Request (`400 INVALID_ARGUMENT`)**
    This is a validation error. It's returned when the request body contains invalid data, such as a missing or incorrectly formatted field.

    ```json
    {
      "error": {
        "code": 400,
        "message": "Credentials are not valid or incomplete. Please check the errors and try again, see details for more information.",
        "status": "INVALID_ARGUMENT",
        "details": [
          {
            "@type": "type.googleapis.com/google.rpc.BadRequest",
            "fieldViolations": [
              {
                "field": "email",
                "description": "Email must be a valid email address"
              },
              {
                "field": "password",
                "description": "Password must not be empty"
              }
            ]
          }
        ]
      }
    }
    ```

* **Not Found (`404 NOT_FOUND`)**
    This error occurs when a requested resource, such as a user or role, does not exist.

    ```json
    {
      "error": {
          "code": 404,
          "message": "User with ID '123' not found.",
          "status": "NOT_FOUND"
      }
    }
    ```

* **Internal Server Error (`500 INTERNAL`)**
    This is a generic error for unexpected server issues. It indicates a problem on the server side and should be reported.

    ```json
    {
      "error": {
          "code": 500,
          "message": "An unexpected error occurred",
          "status": "INTERNAL"
      }
    }
    ```

## 🛡️ Security Best Practices

* **Password Storage:** Passwords are hashed using a strong, modern hashing function like **bcrypt** to ensure they cannot be reversed.
* **Paseto Security:** Paseto tokens are used instead of JWTs to avoid common security vulnerabilities, as they are "platform-agnostic" and explicitly deny unsigned tokens.
* **Input Validation:** All API inputs are meticulously **validated** to prevent malicious data from entering the system.
* **CORS Protection:** The API is configured with **CORS** (Cross-Origin Resource Sharing) to allow requests only from trusted origins.
* **Rate Limiting:** **API rate limiting** is implemented to prevent abuse and protect against denial-of-service attacks.
* **Environment Variables:** All sensitive information, such as database credentials and Paseto secrets, is stored securely in **environment variables**.
* **Structured Error Handling:** Error responses use **Google RPC Status codes** to avoid leaking sensitive internal information to the client.

## ⚡ Performance Optimizations

* **Database Indexing:** Strategic indexes are applied to frequently queried database fields to ensure fast data retrieval.
* **Query Optimization:** All database queries are carefully crafted using the **Squirrel SQL builder** to ensure they are efficient and performant.
* **Pagination:** All endpoints that return lists of data, such as users or roles, are **paginated** to prevent large, slow responses.
* **Caching:** A **Redis cache** is utilized for frequently accessed data to minimize database load and reduce response times.
* **Docker Optimization:** **Multi-stage Docker builds** are used to create small, production-ready container images, reducing deployment time and resource usage.
