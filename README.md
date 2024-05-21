# Online Shopping Microservices Project

This project is an online shopping microservices architecture implemented in Go, using gRPC for communication between services.

## Technologies Used

- **Go (Golang)**: The primary language used for the backend services, providing a fast and efficient runtime environment.
- **gRPC**: Used for communication between microservices, providing a high-performance, language-agnostic RPC framework.
- **Protocol Buffers (protobuf)**: Used for defining service interfaces and data serialization, allowing for efficient communication and easy-to-maintain code.
- **Consul**: Used for service discovery and health checking, enabling dynamic service registration and routing.

## Architecture Overview

The project follows a microservices architecture, with each service responsible for a specific domain or functionality. Some of the key services include:

### `adservice`
Responsible for managing and displaying advertisements.

### `cartservice`
Manages the user's shopping cart. This includes adding, updating, and removing items from the cart.

### `checkoutservice`
Handles the checkout process. This includes verifying cart contents, calculating prices, and processing payment requests.

### `currencyservice`
Provides exchange rate information. This allows the system to convert prices between different currencies, offering accurate pricing information to users worldwide.

### `recommendation Service`
Provides product recommendations.

### `Payment Service`
Processes payment transactions.

### `emailservice`
Handles email communications with users.

### `productcatalogservice`
Manages and provides product catalog information. This includes product details, prices, stock status, and category information.

### `shippingservice`
Handles shipping-related matters. This includes calculating shipping costs and tracking orders.
