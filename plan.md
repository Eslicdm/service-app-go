# Plan for creating pricing-service (Go)

This plan outlines the steps to create a new `pricing-service` in Go, mirroring the structure and best practices observed in the existing `member-service`.

## 1. Project Structure Replication

The `pricing-service` will have a similar directory structure to `member-service`.

```
pricing-service/
├── core/
│   ├── config/
│   ├── entity/
│   └── exception/
├── pricing/
│   ├── controller/
│   ├── dto/
│   ├── repository/
│   └── service/
├── go.mod
├── go.sum
├── main.go
├── .gitignore
├── Dockerfile
├── .env.example
├── .gitattributes
└── docker-compose.yml
```

## 2. Step-by-Step Implementation

### Step 2.1: Create the `pricing-service` root directory - **COMPLETED**

Create the main directory for the new service.

### Step 2.2: Initialize Go Module - **COMPLETED**

Inside `pricing-service`, initialize a new Go module.

```bash
go mod init service-app-go/pricing-service
```

### Step 2.3: Replicate Core Directories - **COMPLETED**

Create the `core` directory and its subdirectories: `config`, `entity`, and `exception`.

### Step 2.4: Replicate Service-Specific Directories - **COMPLETED**

Create the `pricing` directory and its subdirectories: `controller`, `dto`, `repository`, and `service`.

### Step 2.5: Create Essential Files - **COMPLETED**

Create the following files in their respective locations:

*   `pricing-service/main.go`: Initial entry point for the application.
*   `pricing-service/.gitignore`: Copy from `member-service` and adapt if necessary.
*   `pricing-service/Dockerfile`: Copy from `member-service` and adapt for `pricing-service`.
*   `pricing-service/.env.example`: Copy from `member-service` and adapt for `pricing-service`.
*   `pricing-service/.gitattributes`: Copy from `member-service`.
*   `pricing-service/docker-compose.yml`: Create a basic `docker-compose.yml` for local development, similar to `member-service` but for the pricing service.

### Step 2.6: Populate Core Files (Placeholders/Adaptations) - **COMPLETED**

*   **`core/config/`**:
    *   `security_config.go`: Adapt from `member-service/core/config/security_config.go` if security is needed.
*   **`core/entity/`**:
    *   `price.go`: Define the `Price` struct and any related types.
    *   `price_type.go`: Define `PriceType` enum or similar.
*   **`core/exception/`**:
    *   `global_exception_handler.go`: Adapt from `member-service/core/exception/global_exception_handler.go`.

### Step 2.7: Populate Pricing-Specific Files (Placeholders) - **COMPLETED**

*   **`pricing/controller/`**:
    *   `price_controller.go`: Placeholder for API handlers.
*   **`pricing/dto/`**:
    *   `create_price_request.go`: Define DTO for creating prices.
    *   `update_price_request.go`: Define DTO for updating prices.
    *   `update_price_dto.go`: Define DTO for updating prices.
*   **`pricing/repository/`**:
    *   `price_repository.go`: Placeholder for database interaction.
*   **`pricing/service/`**:
    *   `price_service.go`: Placeholder for business logic.

## 3. Initial Content for `main.go` - **COMPLETED**

A basic `main.go` will be created to ensure the service can start.

## 4. Next Steps (Detailed Tasks)

Now that the basic structure is in place, here are the detailed next steps:

### Task 4.1: Define `Price` Entity and `PriceType` Enum - **COMPLETED**

*   Updated `pricing-service/core/entity/price.go` with the `Price` struct (ID, ProductID, Amount, Currency, Type, CreatedAt, UpdatedAt).
*   Updated `pricing-service/core/entity/price_type.go` with `PriceType` enum (e.g., `FREE`, `HALF_PRICE`, `FULL_PRICE`) matching Java structure.

### Task 4.2: Implement `PriceRepository` Interface and MongoDB Implementation - **COMPLETED**

*   Defined `PriceRepository` interface in `pricing-service/pricing/repository/price_repository.go`.
*   Implemented a MongoDB-specific `PriceRepository` in `pricing-service/pricing/repository/mongo_price_repository.go`.
*   Added MongoDB connection setup in `main.go` and `docker-compose.yml`.
*   Updated `.env.example`, `main.go`, and `docker-compose.yml` to use Java-style environment variable names for MongoDB configuration.

### Task 4.3: Implement `PriceService` Business Logic - **COMPLETED**

*   Implemented methods for creating, retrieving, updating, and deleting prices in `pricing-service/pricing/service/price_service.go`.
*   Included basic validation logic.
*   Updated `main.go` to initialize and demonstrate basic usage of the `PriceService`.

### Task 4.4: Implement `PriceController` API Endpoints and Infrastructure Alignment - **COMPLETED**

*   Deleted `member-service/docker-compose.yml` and `pricing-service/docker-compose.yml`.
*   Updated `pricing-service/main.go` to use port `8082`, aligning with the Java pricing service.
*   Updated `pricing-service/pricing/service/price_service.go` to remove `Currency` validation, aligning with `entity.Price`.
*   Modified `pricing-service/core/config/security_config.go` to implement JWKS-based authentication by fetching JWKS from Keycloak, removing reliance on a static `JWT_SECRET`.
*   Updated `service-app-infra/local/docker-compose.yml` to include `KEYCLOAK_REALM_URL` for the Go `pricing-service` and adjusted environment variables for MongoDB connection.
*   Defined REST API endpoints for prices in `pricing-service/pricing/controller/price_controller.go` using Gin.
*   Implemented handlers for CRUD operations.
*   Updated `main.go` to set up the Gin router, initialize the `PriceController`, and register the API routes.
*   **NOTE**: Please run `go mod tidy` in the `pricing-service` directory to fetch the `github.com/gin-gonic/gin`, `github.com/lestrrat-go/jwx/jwk`, and `github.com/lestrrat-go/jwx/jwt` dependencies.

### Explanation on Swagger/OpenAPI Annotations (Go vs. Java)

In Java Spring Boot applications, it's common to use annotations like `@Operation`, `@ApiResponse`, `@Parameter` (from libraries like Springdoc OpenAPI or Swagger-Core) directly within the controller code to generate OpenAPI (Swagger) documentation. These annotations are processed at compile-time or runtime to build the API specification.

In Go, the approach is different:

1.  **No Native Annotation Processing**: Go does not have a native annotation processing mechanism like Java. Therefore, direct annotations on code elements (functions, structs) for OpenAPI generation are not part of the language or standard tooling.
2.  **External Tools**: To generate OpenAPI specifications for Go APIs, external tools are typically used. Popular options include:
    *   **Swag (swag init)**: This tool parses Go comments (specifically formatted comments above handlers, structs, etc.) and generates a `swagger.json` or `swagger.yaml` file. The comments often resemble annotations but are just specially formatted Go comments.
    *   **Go-Swagger (go-swagger)**: This tool can generate server stubs, client libraries, and documentation from an OpenAPI specification file (YAML/JSON). It can also generate an OpenAPI spec from Go code comments.
    *   **Manual Specification**: For simpler APIs or more control, developers might write the OpenAPI specification manually in YAML or JSON.

For this `pricing-service` (Go), the comments added to the `price_controller.go` methods (e.g., `@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`) are intended for use with a tool like `Swag`. These comments are not processed by the Go compiler but are read by `Swag` to generate the OpenAPI documentation. This is the closest equivalent to Java's annotation-based approach for API documentation.

### Task 4.5: Configure Security - **COMPLETED**

*   Adapted `pricing-service/core/config/security_config.go` for JWT authentication/authorization using JWKS.
*   Updated `pricing-service/main.go` to hardcode the Keycloak realm URL, aligning with the `member-service`'s approach.

### Task 4.6: Integrate RabbitMQ for Price Updates - **COMPLETED**

*   Implemented a RabbitMQ consumer in `pricing-service/pricing/messaging/rabbitmq_consumer.go`.
*   Integrated the consumer into `main.go` for initialization, starting, and graceful shutdown.
*   **NOTE**: Please run `go mod tidy` in the `pricing-service` directory to fetch the `github.com/rabbitmq/amqp091-go` dependency.

### Task 4.7: Add Global Exception Handling - **COMPLETED**

*   Implemented global error handling in `pricing-service/core/exception/global_exception_handler.go` with custom error types and structured responses, aligning with `member-service`.
*   Updated `pricing-service/pricing/service/price_service.go` to return these custom error types.
*   Updated `pricing-service/main.go` to load environment variables from `.env.example` using `godotenv.Load()`.
*   Updated `pricing-service/main.go` to use `localhost` for `mongoHost` when running directly from the IDE, and `service-app-infra/local/docker-compose.yml` was cleaned up for the `pricing-service` entry.

### Task 4.8: Write Unit and Integration Tests

*   Create test files for controllers, services, and repositories.

### Task 4.9: Refine Controller and Service Layers with DTOs and Enhanced Error Handling - **COMPLETED**

*   Refactored `pricing-service/pricing/controller/price_controller.go` to use `dto.CreatePriceDTO` and `dto.UpdatePriceDTO` for request bodies.
*   Improved error handling in `price_controller.go` to specifically catch and respond to `exception.InvalidInputError` and `exception.PriceNotFoundError`.
*   Modified `pricing-service/pricing/service/price_service.go` to accept DTOs for `CreatePrice` and `UpdatePrice` methods.
*   Moved ID generation and timestamp setting for new prices from the controller to the `price_service.go`.
*   Ensured validation is performed in the service layer for robustness.

### Task 4.10: Fix ErrorResponse Type and Clarify CreatePriceDTO Usage - **COMPLETED**

*   Defined the `ErrorResponse` struct in `pricing-service/core/exception/global_exception_handler.go` to resolve "Unresolved type 'ErrorResponse'" and "Unknown field 'Message'" errors in the controller.
*   Clarified the necessity of `pricing-service/pricing/dto/create_price_dto.go` despite the repository's upsert capability, emphasizing its role in semantic clarity, distinct validation, and API contract definition.

Please let me know if you'd like me to proceed with "Task 4.8: Write Unit and Integration Tests".