# Using buf build
https://buf.build/

```script
go run github.com/bufbuild/buf/cmd/buf help
```

## if you update `buf.yaml`, should also update `buf.lock`
```script
cd third_party/proto && go run github.com/bufbuild/buf/cmd/buf mod update && cd ../..
```

## NOTE
If tendermint use `BSR: Buf Schema Registry`, can remove `tendermint`
Currently, coping `tendermint-v0.34.19` proto files
See: `go.mod`
