# 2023-11 Activities on "federated star"

We are on the way to implement the feature "federated star / unstar" activity end to end. The goal is to convince the codeberg team to switch this feature on as soon as possible.

At the moment we are implementing the good path. We've reached "create user from response" (see sequence diagram at https://codeberg.org/meissa/forgejo/src/branch/forgejo-federated-star/docs/unsure-where-to-put/threat_analysis_star_activity.md) - so you can expect the first curl-experiment-announcement in near future.

In parallel we start the discussion which new threats might be introduced with this feature. If you are interested in hacking or security, feel welcome to contribute to the threat discussion at: https://codeberg.org/forgejo/forgejo/issues/1854.