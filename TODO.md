- Only allow owner to do challenges
- Allow owner to upload trusted identities
- Long-term identity store
- Improve config
- Improve server.dicePeer to use a single nested stream for simplify?
- Instead of weird generics getter/setter, just use SignOrderedMessage with a proto.Message instead of bytes, and then set it on my own message
- Reduce code duplication and clean up

- Finish cmd and configs. Do new way to setup game servers more generically and which also allows them to have their own configs. For example, factofight shouldnt' require passing in the config like that unless the config is implementing some interface
