# CPSC 559 Group Project - Distributed r/place Clone
Team members: Ahsan Tariq, Jarin Thundathil, Marvellous Chukwukelu, Navpreet Singh, Nour Ajami


## Description
A distributed r/place clone

## Project Structure

We are going to be using the Standard Project Layout (This is a Golang standard layout), its considered the de facto convention for most go projects

```
559-project/
├── cmd/
│   └── main.go                  # Entry point - wires components, starts the server
│
├── internal/
│   ├── canvas/
│   │   └── canvas.go            # Canvas state (2D array, get/set pixel, snapshot)
│   │
│   ├── election/
│   │   └── bully.go             # Modified Bully algorithm, heartbeats (Not implemented yet)
│   │
│   ├── replication/
│   │   └── replication.go       # Broadcast pixel updates to followers (Not implemented yet)
│   │
│   ├── server/
│   │   ├── http.go              # HTTP handlers (GET /snapshot)
│   │   └── websocket.go         # WebSocket server (client connections)
│   │
│   ├── peer/
│   │   └── peer.go              # gRPC client/server (replica-to-replica)
│   │
│   └── snapshot/
│       └── snapshot.go          # Periodic snapshot to disk, recovery in case of failure (not implemented yet)
│
├── proto/
│   └── rplace.proto             # gRPC service definitions 
│
│
├── go.mod
├── go.sum
└── README.md
```
