# Federation Architecture Principles

While implementing federation in forgejo we introduced some conncepts from DomainDrivenDesign:

1. **Aggregate**: Aggregates are clusters of objects (entities or values) which are handled atomic when it comes to persistence.
2. **Validation**: Every object should express it's own validity, whenever someone is interested in
   1. we collect as many invalidity information as possible in one shoot - so we return a list of validation issues if there are some.
   2. Objects entering the lifetime are checked for validity on the borders (after loaded from and before stored to DB, after being newly created (New* functions) or after loaded via web / REST).

Objects in forgefed package reflect Objects from ap or f3 lib but add some Forgejo specific enhancements like more specific validation.

## Federation Model


```mermaid
classDiagram
  namespace activitypub {
    class Activity {
      ID ID
      Type ActivityVocabularyType // Like
      Actor Item
      Object Item
    }
    class Actor {
      ID
      Type ActivityVocabularyType // Person
      Name NaturalLanguageValues
      PreferredUsername NaturalLanguageValues
      Inbox Item
      Outbox Item
      PublicKey PublicKey
    }
  }

  namespace forgfed {
    class ForgePerson {
        Validate() []string
    }
    class ForgeLike {
      Actor PersonID
      Validate() []string
    }
    class ActorID {
      ID               string
      Schema           string
      Path             string
      Host             string
      Port             string
      Source           string
      UnvalidatedInput string
      Validate() []string
    }
    class PersonID {
      AsLoginName() string // "ID-Host"
      AsWebfinger() string // "@ID@Host"
      Validate() []string
    }
    class RepositoryID {
      Validate() []string
    }
    class FederationHost {
      <<Aggregate Root>>
      ID int64
      HostFqdn string
      Validate() []string 
    }

    class NodeInfo {
      Source string
      Validate() []string
    }
  }

  Actor <|-- ForgePerson
  Activity <|-- ForgeLike
  
  ActorID <|-- PersonID
  ActorID <|-- RepositoryID
  ForgeLike *-- PersonID: Actor
  ForgePerson -- PersonID: links to
  FederationHost *-- NodeInfo

  namespace user {
    class User {
      <<Aggregate Root>>
      ID                     int64
      LowerName              string
      Name                   string
      Email                  string
      Passwd                 string
      LoginName              string
      Type                   UserType
      IsActive               bool
      IsAdmin                bool
      NormalizedFederatedURI string
      Validate()             []string
    }

    class FederatedUser {
      ID         int64
      UserID     int64
      ExternalID   string
      FederationHost int64
      Validate() []string
    }
  }

  namespace repository {
    class Repository {
      <<Aggregate Root>>
      ID        int64      
    }

    class FollowingRepository {
      ID             int64
      RepositoryID   int64
      ExternalID     string
      FederationHost int64
      Validate()     []string
    }
  }

  User "1" *-- "1" FederatedUser: FederatedUser.UserID
  PersonID -- FederatedUser : mapped by PersonID.ID == FederatedUser.externalID & FederationHost.ID
  PersonID -- FederationHost : mapped by PersonID.Host == FederationHost.HostFqdn
  FederatedUser -- FederationHost 

  Repository "1" *-- "n" FollowingRepository: FollowingRepository.RepositoryID
  FollowingRepository -- FederationHost
```

## Normalized URI used as ID

In order to use URIs as ID we've to normalize URIs.

A normalized user URI looks like: `https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/1`

In order to normalize URIs we care:

1. Case (all to lower case): `https://federated-REPO.prod.meissa.de/api/v1/activitypub/user-id/1`
2. No relative path: `https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/../user-id/1`
3. No parameters: `https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/1?some-parameters=1`
4. No Webfinger: `https://user1@federated-repo.prod.meissa.de` (with following redirects)
5. No default api: `https://federated-repo.prod.meissa.de/api/activitypub/user-id/1`
6. No autorization: `https://user:password@federated-repo.prod.meissa.de/api/v1/activitypub/user-id/1`
7. No default ports: `https://federated-repo.prod.meissa.de:443/api/v1/activitypub/user-id/1`
8. Accept non default ports: `http://localhost:3000/api/v1/activitypub/user-id/1`
