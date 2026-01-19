# Identity Store

This feature establishes a trusted identity store. We should cache identities in an SQLite DB instead of just in-memory. Then, the server owner can set identities as known/trusted. We should continue adding any challengers to the DB but the owner can mark as trusted or manually add them.

## Requirements
- [ ] Create SQLite DB for identities
- [ ] Store identities here
- [ ] Allow owner to manually add with protected GPRC method
- [ ] Allow owner to list identities and mark some as trusted
