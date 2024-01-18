# Activity for federated star action

- [Activity for federated star action](#activity-for-federated-star-action)
  - [Status](#status)
  - [Context](#context)
  - [Decision](#decision)
  - [Choices](#choices)
    - [1. Map to plain forgejo User](#1-map-to-plain-forgejo-user)
    - [2. Map to User-\&-ExternalLoginUser](#2-map-to-user--externalloginuser)
    - [3. Map to User-\&-FederatedUser](#3-map-to-user--federateduser)
    - [4. Map to new FederatedPerson and introduce a common User interface](#4-map-to-new-federatedperson-and-introduce-a-common-user-interface)


## Status

Still in discussion

## Context

While implementing federation we have to represent federated persons on a local instance.

A federated person should be able to execute local actions (as if he was a local user), ideally without too many code changes.

For being able to map the federated person reliable, the local representation has to carry a clear mapping to the original federated person.

We get actor information as `{"actor": "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",}`. To find out whether this user is available locally without dereference the federated person every time is important for performance & system resilience.

## Decision

tbd

## Choices

### 1. Map to plain forgejo User

1. We map PersonId AsLoginName() (e.g. 13-some.instan.ce) to User.LoginName. Due to limitations of User.LoginName validation mapping may be affected by invalid characters.
2. Created User is limited:
   1. non functional email is generated, email notification is false. At the moment we have problems with email whitelists at this point.
   2. strong password is generated silently
   3. User.Type is UserTypeRemoteUser
   4. User is not Admin
   5. User is not Active

We can use forgejo code (like star / unstar fkt.) without changes.

No new model & persistence is introduced.

But we use fields against their semantic and see some problems / limitations for mapping arise.

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
      <<Aggragate Root>>
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
  PersonID -- Actor: links to
```

### 2. Map to User-&-ExternalLoginUser

1. We map PersonId.AsWebfinger() (e.g. 13@some.instan.ce) to ExternalLoginUser.ExternalID. LoginSourceID may be left Empty.
2. Created User is limited:
   1. non functional email is generated, email notification is false.
   2. strong password is generated silently
   3. User.Type is UserTypeRemoteUser
   4. User is not Admin
   5. User is not Active
3. Created ExternalLoginUser is limited
   1. Login via fediverse is not intended and will not work

We can use forgejo code (like star / unstar fkt.) without changes.

No new model & persistence is introduced, no need for refactorings.

But we use fields against their semantic (User.EMail, User.Password, User.LoginSource, ExternalLoginUser.Login*) and see some problems / limitations for login functionality arise.

Mapping may be more reliable compared to option 1.

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
      AsWebfinger() string // "ID@Host"
    }
  }

  ActorID <|-- PersonID
  ForgeLike *-- PersonID: ActorID
  PersonID -- Actor: links to

  namespace user {
    class User {
      <<Aggregate Root>>
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
      <<Aggregate Root>>
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
  PersonID -- ExternalLoginUser: mapped by AsLoginName() == ExternalID
```

### 3. Map to User-&-FederatedUser

1. We map PersonId.asWbfinger() to FederatedPerson.ExternalID (e.g. 13@some.instan.ce).
2. Created User is limited:
   1. non functional email is generated, email notification is false.
   2. strong password is generated silently
   3. User.Type is UserTypeRemoteUser
   4. User is not Admin
   5. User is not Active

We can use forgejo code (like star / unstar fkt.) without changes.

Introduce FederatedUser as new & persistence, no need for refactorings.

But we use fields (User.EMail, User.Password) against their semantic, but we probably can handle the problems arising.

We will be able to have a reliable mapping.

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
      <<Aggregate Root>>
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

    class FederatedUser {
      ID         int64
      UserID     int64
      RawData    map[string]any
      ExternalID   string
      FederationHost int64
    }
  }
  User *-- FederatedUser: FederatedUser.UserID
  PersonID -- FederatedUser : mapped by PersonID.asWebfinger() == FederatedUser.externalID

  namespace forgefed {
    
    class FederationHost {
      <<Aggregate Root>>
      ID int64
      HostFqdn string
    }

    class NodeInfo {
      Source string
    }
  }
  FederationHost *-- NodeInfo
  FederatedUser -- FederationHost

 
```

### 4. Map to new FederatedPerson and introduce a common User interface

Cached FederatedPerson is mainly independent to existing User. At every place of interaction we have to enhance persistence & introduce a common User interface.

1. We map PersonId.asWbfinger() to FederatedPerson.ExternalID (e.g. 13@some.instan.ce).
2. We will have no semantic mismatch.

We can use forgejo code (like star / unstar fkt.) after refactorings only.

We introduce new model & persistence.

We will be able to have a reliable mapping.

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
  PersonID -- Actor: links to

  namespace user {
    class CommonUser {
      <<Interface>>
    }
    class User {
      
    }    
  }
  User ..<| CommonUser 
  
  namespace forgefed {
    class FederatedPerson {
      <<Aggregate Root>>
      ID         int64
      UserID     int64
      RawData    map[string]any
      ExternalID   string
      FederationHost int64
    }
    
    class FederationHost {
      <<Aggregate Root>>
      ID int64
      HostFqdn string
    }

    class NodeInfo {
      Source string
    }
  }
  PersonID -- FederatedPerson : mapped by PersonID.asWebfinger() == FederatedPerson.externalID
  FederationHost *-- NodeInfo
  FederatedPerson -- FederationHost
  FederatedPerson ..<| CommonUser  
```