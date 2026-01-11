# Example Keys

## ⚠️ SECURITY WARNING

These keys are for **DEVELOPMENT AND TESTING ONLY**. Never use them in production!

## Usage

- `example_ed25519.pem`: Example private key for testing
- `example_ed25519.pub.pem`: Corresponding public key

## Generate New Keys

```bash
go run main.go --generate-key --output keys/my_server.pem
```

## Key Detection

The system will warn you if example keys are detected in non-testing environments.