# Activity for federated star action

## Status

Still in discussion

## Context

While implementing federation we have to represent persons federated to a local instance. A federated person should be able to execute local actions (as it was a local user) without to many code changes. But the federated person should be able to map to the origin person and keep the crypto stuff to ensure action integrity.

## Decision

tbd

## Choices

### 1. Map to User.LoginName by AsLoginName()

1. We map PersonId AsLoginName() to User.LoginName.
2. We accept only URIs as Actor Items
3. We can lookup for federated users without fetching the Person every time.
4. Created User is limited:
   1. non functional email is generated, email notification is false.
   2. strong password is generated silently
   3. User.Type is UserTypeRemoteUser
   4. User is not Admin
   5. User is not Active

```mermaid
classDiagram
  namespace activitypub {
    class ForgeLike {
      ID ID
      Type ActivityVocabularyType // Like
      Actor Item
      Object Item
    }
    class Actor {
      ID
      URL Item
      Type ActivityVocabularyType // Person
      Name NaturalLanguageValues
      PreferredUsername NaturalLanguageValues
      Inbox Item
      Outbox Item
      PublicKey PublicKey
    }
    class ActorID {
      ID               string
      Source           string
      Schema           string
      Path             string
      Host             string
      Port             string
      UnvalidatedInput string
    }
    class PersonID {
      AsLoginName() string // "ID-Host"
    }
  }

  ActorID <|-- PersonID
  ForgeLike *-- PersonID: ActorID

  namespace forgejo {
    class User {
      ID        int64 
      LowerName string
      Name      string
      Email     string
      Passwd    string
      LoginName   string
      Type        UserType
      IsActive bool
      IsAdmin bool
    }
  }

  PersonID -- User: mapped by AsLoginName() == LoginName
```

### 2. Map to ExternalLoginUser

Would improve the ability to map to the federation source.

But login Propagation stuff is not going to be used and will maybe be harmful.

```mermaid
classDiagram
  namespace activitypub {
    class ForgeLike {
      ID ID
      Type ActivityVocabularyType // Like
      Actor Item
      Object Item
    }
    class Actor {
      ID
      URL Item
      Type ActivityVocabularyType // Person
      Name NaturalLanguageValues
      PreferredUsername NaturalLanguageValues
      Inbox Item
      Outbox Item
      PublicKey PublicKey
    }
    class ActorID {
      ID               string
      Source           string
      Schema           string
      Path             string
      Host             string
      Port             string
      UnvalidatedInput string
    }
    class PersonID {
      AsLoginName() string // "ID-Host"
    }
  }

  ActorID <|-- PersonID
  ForgeLike *-- PersonID: ActorID

  namespace user {
    class User {
      ID        int64
      LoginSource int64
      LowerName string
      Name      string
      Email     string
      Passwd    string
      LoginName   string
      Type        UserType
      IsActive bool
      IsAdmin bool
    }

    class ExternalLoginUser {
      ExternalID        string
      LoginSourceID     int64
      RawData           map[string]any
      Provider          string        
    }
  }

  namespace auth {
    class Source {
      ID            int64
      Type          Type
      Name          string  
      IsActive      bool           
      IsSyncEnabled bool  
    }
  }

  User *-- ExternalLoginUser: ExternalLoginUser.UserID
  User -- Source
  ExternalLoginUser -- Source  
```

### 3. Map to FederatedUser

Would improve the ability to map to the federation source.

```mermaid
classDiagram
  namespace activitypub {
    class ForgeLike {
      ID ID
      Type ActivityVocabularyType // Like
      Actor Item
      Object Item
    }
    class Actor {
      ID
      URL Item
      Type ActivityVocabularyType // Person
      Name NaturalLanguageValues
      PreferredUsername NaturalLanguageValues
      Inbox Item
      Outbox Item
      PublicKey PublicKey
    }
    class ActorID {
      ID               string
      Source           string
      Schema           string
      Path             string
      Host             string
      Port             string
      UnvalidatedInput string
    }
    class PersonID {
      AsLoginName() string // "ID-Host"
      AsWebfinger() string // "@ID@Host"
    }
  }

  ActorID <|-- PersonID
  ForgeLike *-- PersonID: ActorID

  namespace user {
    class User {
      ID        int64
      LowerName string
      Name      string
      Email     string
      Passwd    string
      LoginName   string
      Type        UserType
      IsActive bool
      IsAdmin bool
    }
  }

  namespace forgefed {
    class FederatedUser {
      ID         int64
      UserID     int64
      RawData    map[string]any
      RemoteID   string
      RemoteInfo int64
    }
    class FederationInfo {
      ID int64
      HostFqdn string
      NodeInfo NodeInfo
    }
  }

  User o-- FederatedUser: FederatedUser.UserID
  FederatedUser -- FederationInfo
  PersonID -- FederatedUser : maped by PersonID.ID == FederatedUser.RemoteID
```
