# r/place Clone

This project is a full-stack implementation of a r/place-like collaborative pixel art canvas. Users can place colored pixels on a shared grid, creating a dynamic and interactive community artwork.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
- [Development](#development)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

## Architecture Overview

The project follows a microservices architecture with the following components:

1. **Frontend**: React-based web application
2. **Backend**: Rust-based WebSocket server for real-time updates
3. **Auth Service**: Go-based authentication service
4. **Redis**: Used for caching and data persistence
5. **Nginx**: Reverse proxy and load balancer

The services are containerized using Docker and orchestrated with Docker Compose.

## Features

- Real-time collaborative pixel placement
- Google OAuth2 authentication
- Efficient grid updates using WebSockets
- Quadrant-based rendering for improved performance
- Color picker for pixel placement
- User session management

## Tech Stack

- Frontend: React, Bootstrap, @react-oauth/google
- Backend: Rust (Actix-web, tokio)
- Auth Service: Go (Gin framework)
- Database: Redis
- Reverse Proxy: Nginx
- Containerization: Docker
- Orchestration: Docker Compose

## Getting Started

1. Clone the repository:
   ```
   git clone https://github.com/your-username/rplace-clone.git
   cd rplace-clone
   ```

2. Set up environment variables:
   - Create a `.env` file in the root directory
   - Add the following variables:
     ```
     JWT_SECRET=your_jwt_secret
     REACT_APP_GOOGLE_CLIENT_ID=your_google_client_id
     REACT_APP_API_BASE_URL=http://localhost:8081
     ```

3. Build and run the project using Docker Compose:
   ```
   docker-compose up --build
   ```

4. Access the application at `http://localhost:80`

## Development

### Frontend

The frontend is a React application located in the `frontend` directory. To run it locally:

1. Navigate to the `frontend` directory
2. Install dependencies: `npm install`
3. Start the development server: `npm start`

### Backend

The backend is a Rust application located in the `backend` directory. To run it locally:

1. Navigate to the `backend` directory
2. Build and run the application: `cargo run`

### Auth Service

The auth service is a Go application located in the `auth` directory. To run it locally:

1. Navigate to the `auth` directory
2. Build and run the application: `go run main.go`

## Deployment

The project is designed to be deployed using Docker Compose. Follow these steps for production deployment:

1. Update the `docker-compose.yaml` file with production-specific configurations
2. Set up SSL certificates for secure communication
3. Update the Nginx configuration in `frontend/nginx.conf` for production use
4. Build and deploy the containers:
   ```
   docker-compose -f docker-compose.yaml up -d
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
