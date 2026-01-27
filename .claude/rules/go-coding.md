---
paths: "**/*.go"
---

# Go Coding Rules

## Error Handling

Wrap errors with context using `%w`:
```go
// WRONG
return nil, err

// CORRECT
return nil, fmt.Errorf("load preset %s: %w", name, err)
```

## Context Propagation

ALWAYS pass `context.Context` as first parameter:
```go
// CORRECT
func (s *Service) Fetch(ctx context.Context, id string) (*Data, error)

// WRONG: context.Background() in library code
func (s *Service) Fetch(id string) (*Data, error) {
    return s.client.Get(context.Background(), id)
}
```

## Naming
```go
// Packages: short, singular, lowercase
package race     // not "races" or "raceUtils"

// Exported: descriptive, don't stutter
race.ID          // not race.RaceID

// Acronyms: consistent case
userID, apiClient, HTTPServer
```

## Interface Design

Accept interfaces, return structs. Define at consumer side:
```go
// CORRECT: Easy to test
type raceGetter interface {
    Get(ctx context.Context, id string) (*Race, error)
}
type Service struct { races raceGetter }

// WRONG: Hard to test
type Service struct { races *PostgresRepository }
```