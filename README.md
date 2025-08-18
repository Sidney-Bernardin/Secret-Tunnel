# Secret Tunnel

## Example
```bash
# Build
go build -o secret-tunnel main.go

# Run
export SECRET_TUNNEL_POSTGRES_URL="..."
find ./src -name '*.yaml' | xargs ./secret-tunnel > out.yaml

# Run with Docker
docker build -t secret-tunnel ./Docker
docker run \
    -e SECRET_TUNNEL_POSTGRES_URL="..." \
    -v ./local-src:/secret-tunnel/src \
    secret-tunnel > out.yaml
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
