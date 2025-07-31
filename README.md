# Secret Tunnel

## Example
```bash
# Build
go build -o secret-tunnel main.go

# Run
find ./src -name "*.yaml" | xargs ./secret-tunnel > out.yaml
```

## Usage
```
./secret-tunnel [FLAG]... [FILE]...
```

### Environment Variables
* SECRET_TUNNEL_POSTGRES_URL
  > URL for the Postgres database.

### Flags
* -single-quote
  > Single or double quotes for strings.
