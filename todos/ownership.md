# Ownership

This feature establishes trust between the server and the owner (person). This should work using PK auth where the server has a configuration for the owner's public key. The owner then sends signed messages to the server which will have certain functions that are restricted to the owner only.

## Requirements
- [x] Update config to know owner's PK
  - [x] Update config.cue
  - [x] Update tests
  - [x] Verify test data using 'task validate_config_testdata'
- [x] Establish a format for communicating the trusted messages (probably use existing SignedMessage format)
- [x] Update protos for these messages
- [x] Protect the Challenge endpoint
- [x] Update tests for server/factorfight to show an error authorizing
- [x] Create CLI at cmd/challenge/main.go using "github.com/urfave/cli/v3" to easily create, sign, and send a challenge message. Update Taskfile to use it

## Nice to have (research)
- [ ] Create a middleware/interceptor on the server for protected methods
- [ ] Can we add something to protos that specifies it is a protected method? Can this auto-wire to a custom middleware?
