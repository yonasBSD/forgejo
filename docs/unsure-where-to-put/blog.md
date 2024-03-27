# 2024-03 Ui-Work for Sending Like Activities

You can find this post here: https://domaindrivenarchitecture.org/posts/2024-03-27-state-of-federation/

# 2024-02 Considerations on Mapping and Architectural decisions

This month we discussed how a federated Person should be mapped to a local FederatedUser representation. Having a reliable mapping will be very important to trace code- / issue- and other ownerships.

I am very glad about the constructive & good discussion and many cool inputs. If you are interested in the federation related architecture you can have a sneak preview here: https://codeberg.org/meissa/forgejo/src/branch/forgejo-federated-star/docs/unsure-where-to-put/federation-architecture.md

Next an final step on our way to "federated Stars" will be "UI star triggers a federated Like Activity (in case of mirrored repos?)" - stay tuned for next month step :-).

In case of interest find the current roadmap at: https://codeberg.org/forgejo/forgejo/pulls/1680

# 2024-01 Federated staring with Like Activity

We did the next step. We now use a plain Like Activity for expressing the Star action.
In addition we fixed some bugs, made error responses more meaningful, improved security by validating every input we get on federation & mitigate identified threats (SlowLories, Replay Attacks, Block by future StartTime).

DOS attacks we now mitigate in our k8s ingress. Find the code in our [PR for c4k-forgejo](https://repo.prod.meissa.de/meissa/c4k-forgejo/pulls/3).

At https://federated-repo.prod.meissa.de/me/star-me you can try out the current code the same way as described above with the following activity (maybe find an unused user by alternating the actors user-id).

``` json
{
  "id": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/1/outbox/12",
  "type": "Like",
  "actor": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/12",
  "object": "https://federated-repo.prod.meissa.de/api/v1/activitypub/repository-id/1",
  "startTime": "2024-01-05T23:00:00-08:00"
}
```

Please consider to increment the `startTime` for each api-request - maybe use the current time is a good idea.

In case of interest find the current roadmap at: https://codeberg.org/forgejo/forgejo/pulls/1680

# 2023-12 Federated staring open for test

Hey, we ar on our way to implement federated stars. We created a test instance to show the new feature - an now you can test federation live :-)

1. **The repo** ready to receive your star is located at: https://federated-repo.prod.meissa.de/me/star-me
2. **Post a star activity** at: https://federated-repo.prod.meissa.de/api/swagger#/activitypub/activitypubRepository & press the `Try It Out`` button. The input can look like: ![star-via-api.png](star-via-api.png)
3. Put "1" in to the repo & add the following payload   
    ``` json
    {
      "id": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/1/outbox/12",
      "type": "Star",
      "source": "forgejo",
      "actor": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/14",
      "object": "https://federated-repo.prod.meissa.de/api/v1/activitypub/repository-id/1"
    }
    ```
4. As every user can only put one star, we created 12 users for your experiment on our instance `"actor": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/2-13",`. But if you are on a forgejo instance having active `activitypub/user-id` api you can insert also your foreign-instance-user-uri here.
5. Press execute & visit again the repo (https://federated-repo.prod.meissa.de/me/star-me) and enjoy your star :-) ![find-your-new-star](find-your-new-star.png)

At the moment we discuss threats arising by this feature. If you are interested we will be happy to get your 2 cents here: https://codeberg.org/forgejo/forgejo/issues/1854

# 2023-11 Activities on "federated star"

We are on the way to implement the feature "federated star / unstar" activity end to end. The goal is to convince the codeberg team to switch this feature on as soon as possible.

At the moment we are implementing the good path. We've reached "create user from response" (see sequence diagram at https://codeberg.org/meissa/forgejo/src/branch/forgejo-federated-star/docs/unsure-where-to-put/threat_analysis_star_activity.md) - so you can expect the first curl-experiment-announcement in near future.

In parallel we start the discussion which new threats might be introduced with this feature. If you are interested in hacking or security, feel welcome to contribute to the threat discussion at: https://codeberg.org/forgejo/forgejo/issues/1854.
