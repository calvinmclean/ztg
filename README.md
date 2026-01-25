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

## Challenging Other Players

Once your server is deployed, you can challenge other players:

```bash
ztg challenge send \
  --target ztg.fly.dev:443 \
  --game ztg.FactorFight.v1 \
  --server ztg.fly.dev:443 \
  --key-path .keys/owner.pem
```

Replace `--server ztg.fly.dev:443` with your deployed server address to challenge me with your implementation.
