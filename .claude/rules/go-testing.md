---
paths: "**/*.go"
---

# Go Testing Rules

## Coverage

```bash
task test           # Run tests with coverage
task test:coverage  # Check coverage threshold
```

## Table-Driven Tests (Preferred)

```go
func TestParseQuantType(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid Q4_K_M", "Q4_K_M", "Q4_K_M", false},
        {"lowercase", "q4_k_m", "Q4_K_M", false},
        {"invalid", "Q9_X", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseQuantType(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Mocking

Use interfaces for dependencies that need mocking:

```go
type ProcessManager interface {
    Start(ctx context.Context, args []string) error
    Stop(ctx context.Context) error
    IsRunning() bool
}
```
