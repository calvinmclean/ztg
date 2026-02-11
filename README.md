# Zero Trust Gaming (ztg)

A Go framework for building peer-to-peer (no centralized server) games with cryptographic security guarantees. This project implements zero-trust gaming principles where no participant needs to trust any other participant.

Implement and deploy your own Game Strategies to challenge other players!

## Overview

`ztg` provides:
- **Cryptographically secure dice rolling** with peer-to-peer commitment schemes
- **Identity verification** using Ed25519 signatures
- **Tamper-proof game state** through hash chains
- **gRPC-based server architecture** with mutual authentication

## Features

### Secure Dice Rolling
- Uses commitment schemes to prevent cheating in distributed dice rolls
- Cryptographically random secrets with SHA256 hash verification
- Peer-to-peer communication without trusted third parties
- Supports rolling up to 8 dice simultaneously

### Identity & Authentication
- Ed25519 key pairs for cryptographic identity
- All RPC calls require valid signatures
- Server signs all responses with private key
- Hash chain verification for game state integrity

### Games
- **Factor Fight**: Mathematical strategy game where players combine dice rolls to reach the target 101 with two pawns
- **High Roll**: Dice rolling game with distributed randomness

## Quick Start

### Installation
```bash
go install ztg/cmd/ztg@latest
```

### Generate Keys
```bash
ztg key generate --name my_key
```

## Building Your Own Server

The best way to get started is by implementing your own server with a custom strategy and game completion handler. See the [example/server](./example/server/) directory for a complete reference implementation.

### Key Components

1. **Strategy**: Implement your game strategy logic
2. **OnGameComplete**: Handle game completion events (notifications, logging, etc.)
3. **Configuration**: Use CUE to define required server settings

### Configuration with CUE

Create a `config.cue` file based on the schema in [config/config.cue](./config/config.cue):

```cue
package config

server: {
    port: 8080
    log_level: "info"
}

identity: {
    server_name: "my-server"
    owner_name: "my-user"
    owner_public_key_file: ".keys/owner.pub"
    private_key_file: ".keys/server.pem"
    server_address: "my-server.example.com:443"
}
```

### Example Implementation

The example server shows:
- Custom strategy implementation using `factorfight.DefaultStrategy`
- Pushover notifications on game completion
- Proper key management and server setup

```go
ffCfg := ffserver.Config{
    Strategy: factorfight.DefaultStrategy,
    OnGameComplete: func(win bool, _ factorfight.GameLog) {
        // Your custom game completion logic here
        log.Printf("Game completed. Win: %v", win)
    },
}
```

## Identity Store & Trust Management

ztg now includes persistent identity storage using PostgreSQL. This allows server owners to maintain a trusted identity database and manage trust relationships.

### How It Works

The identity store provides:

- **Persistent Storage**: Identities are stored in PostgreSQL
- **Trust Management**: Server owners can mark identities as trusted or untrusted
- **Automatic Discovery**: New challengers are automatically added to the database
- **Local Verification**: Trusted identities can be verified without HTTP requests

### Running with Storage
```bash
# Start PostgreSQL with Docker Compose
docker-compose up -d postgres

# Run server with PostgreSQL connection
ZTG_DATABASE_URL="postgres://ztg:password@localhost:5432/ztg?sslmode=disable" \
go run cmd/ztg/main.go server
```

### Managing Trust Status

#### List All Identities
```bash
ztg identity list --server localhost:50052
```

#### Mark Identity as Trusted
```bash
ztg trust send \
  --server localhost:50052 \
  --peer localhost:50053 \
  --trusted \
  --key-path <owner_key_path> 
```

### Local vs Remote Player Challenges

The identity store enables a powerful use case where Player A runs locally and challenges Player B's server:

1. **Player A** runs a local server

2. **Player B** runs their deployed server with storage

3. **Player B** registers their identity with Player A's local server:
   ```bash
   # First, get Player B's public key (outputs in base64 format)
   ztg key show --public --key-path .keys/server.pem
   
   # Then add it to Player A's local server
   ztg identity add \
     --server localhost:50052 \
     --peer player-b.example.com:443 \
     --peer-name "Player B Server" \
     --owner-name "Player B" \
     --public-key [PLAYER_B_PUBLIC_KEY]
   ```

4. **Player A** challenges Player B:
   ```bash
   ztg challenge send \
     --target player-b.example.com:443 \
     --game ztg.FactorFight.v1 \
     --server localhost:50052 \
     --key-path <owner_key_path> 
   ```

5. **Player B** now knows Player A's identity without making HTTP requests because:
   - Player A's identity was stored in Player B's database during the first challenge
   - Player B can verify Player A's identity locally using the cached public key
   - Future challenges from Player A can be verified instantly without network calls

This enables efficient zero-trust gaming while maintaining identity persistence across server restarts.

## Database Migrations

The project uses [golang-migrate](https://github.com/golang-migrate/migrate) to manage database schema changes. Migrations are located in the `migrations/` directory.

#### Installation

```bash
# Install CLI tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Running Migrations with golang-migrate

**Run Up Migrations**
```bash
# Up migrations (apply all pending migrations)
migrate -database "postgres://user:password@localhost:5432/ztg_db?sslmode=disable" \
        -path "migrations" \
        up

# Using environment variable
migrate -database "$DATABASE_URL" -path "migrations" up
```

**Check migration status**:
```bash
migrate -database "$DATABASE_URL" -path "migrations" version
```

**Rollback migrations**:
```bash
# Rollback one migration
migrate -database "$DATABASE_URL" -path "migrations" down 1

# Rollback all migrations
migrate -database "$DATABASE_URL" -path "migrations" down
```

## Challenging Other Players

Once your server is deployed, you can challenge other players:

```bash
ztg challenge send \
  --target ztg.fly.dev:443 \
  --game ztg.FactorFight.v1 \
  --server ztg.fly.dev:443 \
  --key-path <owner_key_path> 
```

Replace `--server ztg.fly.dev:443` with your deployed server address to challenge me with your implementation.
