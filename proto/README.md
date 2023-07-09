# Using buf build
https://buf.build/

```script
go run github.com/bufbuild/buf/cmd/buf help
```

## if you update `buf.yaml`, should also update `buf.lock`
```script
cd proto && go run github.com/bufbuild/buf/cmd/buf mod update && cd ..
```
