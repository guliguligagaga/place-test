# r/place Clone System

## Overview

This system is a scalable, real-time implementation of a collaborative pixel art canvas, inspired by Reddit's r/place experiment. It allows multiple users to simultaneously place colored pixels on a shared canvas, creating a dynamic, community-driven artwork.

Hosted on my Pi4 https://grid.guliguli.work

![Untitled-2024-07-22-2203](https://github.com/user-attachments/assets/4aa36f43-9dc0-4e7d-86ce-7d9183ff74f5)


## Architecture

The system is built using a microservices architecture, deployed on a K3s Kubernetes cluster, with the following components:

1. **React Client**: User-facing web application
2. **Cloudflare**: CDN and security layer
3. **Traefik**: Ingress controller and request router
4. **Auth Service**: User authentication and authorization
5. **Draw Service**: Processes pixel placement requests
6. **Kafka**: Message queue for pixel updates
7. **Grid Service**: Processes updates and manages canvas state
8. **Redis**: In-memory data store for current canvas state
9. **WebSocket Service**: Real-time updates to connected clients

## Data Flow

1. Users interact with the React client via HTTPS/WSS
2. Requests pass through Cloudflare for security and optimization
3. Traefik routes requests to appropriate services within the K3s cluster
4. Draw requests are validated, processed, and sent to Kafka
5. Grid Service consumes Kafka messages and updates Redis
6. WebSocket Service pushes updates to connected clients

## Key Features

- Real-time collaborative editing
- Scalable architecture to handle high concurrent user loads
- Efficient canvas state management using Redis
- Secure and optimized communication via Cloudflare
- Authentication and authorization for user management

## Deployment

The system is designed to be deployed on a K3s Kubernetes cluster.

## Development Setup

TODO
