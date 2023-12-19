```
# edn notation
{@context [
    "as":    "https://www.w3.org/ns/activitystreams#",
    "forge": "https://forgefed.org/ns#",],
  ::as/id "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1/outbox/12345",
  ::as/type "Star",
  ::forge/source "forgejo",
  ::as/actor "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",
  ::as/object "https://codeberg.org/api/v1/activitypub/repository-id/12"
}

# json notation
{"id": "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1/outbox/12345",
  "type": "Star",
  "source": "forgejo",
  "actor": "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",
  "object": "https://codeberg.org/api/v1/activitypub/repository-id/1"
}
```

This way of expressing stars will have the following features:

1. Actor & object may be dereferenced by (ap-)api
2. The activity can be referenced itself (e.g. in order to express a result of the triggered action)
3. Star is a special case of a Like. Star only happens in ForgeFed context. Different things should be named differnt ...
4. With the `source` given it would be more easy to distinguish the uri layout for object and actor id's and make implementation more straight forward
   1. The `source` field reflects the software sending an activity. Values of may be forgejo, gitlab, ...
   2. Knowing the sending system will it make easier to interact with:
      1. We know exactly how the actor can be derefernced (see: https://codeberg.org/meissa/forgejo/src/commit/7cac9806f8247963b1cdce3f2c5f5d1bc3763fbe/models/forgefed/actor.go#L121) 
      2. We know how we can validate the given references (see: https://codeberg.org/meissa/forgejo/src/commit/7cac9806f8247963b1cdce3f2c5f5d1bc3763fbe/routers/api/v1/activitypub/repository.go#L180)

See also:
1. [spec in clojure]: https://repo.prod.meissa.de/meissa/activity-pub-poc/src/branch/forgefed_star/src/test/cljc/org/domaindrivenarchitecture/fed_poc/forgefed_test.cljc#L36-L41

