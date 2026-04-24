# CPSC 559 Group Project - Distributed r/place Clone
Team members: Ahsan Tariq, Jarin Thundathil, Marvellous Chukwukelu, Nour Ajami


## Description
A distributed r/place clone

## Project Structure

We are going to be using the Standard Project Layout (This is a Golang standard layout), its considered the de facto convention for most go projects

```
559-project/
├── cmd/
│   ├── main.go
│   ├── reuse_unix.go
│   └── reuse_windows.go
│
├── internal/
│   ├── canvas/
│   │   └── canvas.go
│   ├── config/
│   │   └── env.go
│   ├── models/
│   │   └── pixels.go
│   ├── node/
│   │   ├── election.go
│   │   ├── heartbeat.go
│   │   └── node.go
│   ├── replication/
│   │   └── replication.go
│   ├── server/
│   │   ├── http.go
│   │   └── websocket.go
│   └── snapshot/
│       └── snapshot.go
│
├── logs/
│
├── scripts/
│   ├── kill-node.sh
│   ├── killer.sh
│   ├── restart-node-demo.sh
│   ├── restart-node.sh
│   ├── start-replicas-demo.sh
│   ├── start-replicas.sh
│   └── stop-replicas.sh
│
├── web/
│   └── src/
│       ├── App.css
│       ├── App.js
│       ├── components/
│       │   ├── Canvas.css
│       │   ├── Canvas.js
│       │   ├── HUD.css
│       │   ├── HUD.js
│       │   ├── Toolbar.css
│       │   └── Toolbar.js
│       └── services/
│           └── canvasService.js
│
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## Running the System

### Start all replicas

From the project root:

```bash
./scripts/start-replicas.sh
```

```bash
./scripts/start-replicas.sh
```

This launches all replica nodes (including leader election and replication services).

### Start the frontend

In a separate terminal:

```bash
cd web
npm install
npm start
```

The frontend will be available at:

```text
http://localhost:3000
```

### Access a specific replica (optional)

Connect directly to a particular backend replica:

```text
http://localhost:3000/?server=localhost:8083
```

### Stop replicas

```bash
./scripts/stop-replicas.sh
```


## Running on Physical Machines

Update the peer addresses in the startup (use `start-replicas-demo.sh`) script so each node uses reachable machine IPs instead of `localhost`.

Example format:

```bash
PEERS="2=192.168.1.102:8081,3=192.168.1.103:8082,4=192.168.1.104:8083"
```

Then start a node on each machine:

```bash
PORT=8080 NODE_ID=1 LEADER_ID=0 ./bin/server
```

Example for another machine:

```bash
PORT=8081 NODE_ID=2 LEADER_ID=0 ./bin/server
```

Repeat for each replica, adjusting:

- `PORT`
- `NODE_ID`
- `PEERS`

Then run the frontend and connect to any reachable replica:

```text
http://<machine-ip>:3000/?server=<backend-ip>:8080
```
