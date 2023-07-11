# Using buf build
https://buf.build/

```script
go run github.com/bufbuild/buf/cmd/buf help
```

## buf.yaml
See:
- https://buf.build/docs/lint/overview/
- https://buf.build/docs/lint/rules/

## if you update `buf.yaml`, should also update `buf.lock`
```script
cd proto && go run github.com/bufbuild/buf/cmd/buf mod update && cd ..
```
