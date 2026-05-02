# Secret Tunnel

## Example
```bash
# Build
go build -o secret-tunnel main.go

# Run
find ./src -name '*.yaml' | xargs ./secret-tunnel > out.yaml
```

## Usage
```
./secret-tunnel [FLAG]... [FILE]...
```

### Environment Variables
* ST_AWS_BASE_ENDPOINT
* ST_AWS_DATABASE_SECRET_NAME

### Flags
* -single-quote
  > Use single/double quotes the output. 
