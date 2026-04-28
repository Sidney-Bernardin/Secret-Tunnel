# Secret Tunnel

## Example
```bash
# Build
go build -o secret-tunnel main.go

# Run
export SECRET_TUNNEL_POSTGRES_USERNAME="..."
export SECRET_TUNNEL_POSTGRES_HOST="..."
export SECRET_TUNNEL_POSTGRES_PORT="..."
export SECRET_TUNNEL_POSTGRES_DATABASE="..."
find ./src -name '*.yaml' | xargs ./secret-tunnel > out.yaml

# Run with Docker
docker build -t secret-tunnel ./Docker
docker run \
    -e SECRET_TUNNEL_POSTGRES_URL="..." \
    -e SECRET_TUNNEL_POSTGRES_USERNAME="..." \
    -e SECRET_TUNNEL_POSTGRES_HOST="..." \
    -e SECRET_TUNNEL_POSTGRES_PORT="..." \
    -e SECRET_TUNNEL_POSTGRES_DATABASE="..." \
    -v ./local-src:/secret-tunnel/src \ # Mount your directory of input YAML files.
    secret-tunnel > out.yaml
```

## Usage
```
./secret-tunnel [FLAG]... [FILE]...
```

### Environment Variables
* SECRET_TUNNEL_POSTGRES_USERNAME
* SECRET_TUNNEL_POSTGRES_HOST
* SECRET_TUNNEL_POSTGRES_PORT
* SECRET_TUNNEL_POSTGRES_DATABSE
* SELECT_TUNNEL_AWS_REGION
* SELECT_TUNNEL_AWS_SECRET_POSTGRES_PASSWORD
  > The AWS secret-id of the postgres password.

### Flags
* -single-quote
  > Use single/double quotes the output. 
