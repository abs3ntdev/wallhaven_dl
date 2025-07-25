# Architecture Documentation

## Overview

The wallhaven_dl application has been modernized with a clean architecture pattern that separates concerns and improves maintainability.

## Project Structure

```
├── cmd/                    # Command handlers
│   ├── search.go          # Search command handler
│   ├── previous.go        # Previous wallpaper handler
│   ├── stats.go           # Statistics handler
│   ├── cleanup.go         # Cleanup handler
│   ├── favorites.go       # Favorites management
│   └── rate.go            # Rating handler
├── internal/              # Internal packages
│   ├── config/            # Configuration management
│   ├── constants/         # Application constants
│   ├── errors/            # Custom error types
│   ├── executor/          # Script execution
│   ├── interfaces/        # Dependency injection interfaces
│   └── validator/         # Input validation
├── src/wallhaven/         # Core wallpaper functionality
│   ├── search.go          # API interaction
│   └── cache.go           # Caching system
└── main.go                # Application entry point
```

## Key Improvements

### 1. Modular Architecture
- Separated CLI logic from business logic
- Command handlers for each major function
- Clear separation of concerns

### 2. Dependency Injection
- Interfaces for all major components
- Testable architecture
- Loose coupling between components

### 3. Configuration Management
- Centralized configuration system
- Validation for all inputs
- Default values with override capability

### 4. Error Handling
- Custom error types with context
- Structured error reporting
- Proper error wrapping

### 5. Performance Improvements
- Concurrent download limiting
- HTTP connection pooling
- Efficient cache operations

### 6. Testing
- Unit tests for core functionality
- Interface-based testing
- Configuration validation tests

### 7. Logging
- Structured logging with slog
- Debug mode support
- Contextual log messages

## Usage Examples

### Basic Search
```bash
wallhaven_dl search anime
```

### Advanced Search with Options
```bash
wallhaven_dl search --categories=010 --purity=110 --sort=toplist nature
```

### Favorites Management
```bash
wallhaven_dl favorite add
wallhaven_dl favorite list
wallhaven_dl favorite random
```

### Statistics and Cleanup
```bash
wallhaven_dl stats
wallhaven_dl cleanup --mode=unused --dryRun
```

## Configuration

The application supports environment variables:
- `WH_API_KEY`: Wallhaven API key for authenticated requests
- `DEBUG`: Enable debug logging
- `HOME`: Used for default download path

## Testing

Run tests with:
```bash
go test ./...
```

## Building

Build the application:
```bash
go build -o wallhaven_dl .
```

## Contributing

The modular architecture makes it easy to add new features:

1. Create a new handler in `cmd/`
2. Add any new constants to `internal/constants/`
3. Add validation if needed in `internal/validator/`
4. Wire up the command in `main.go`
5. Add tests for new functionality