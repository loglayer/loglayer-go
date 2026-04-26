```go
log = log.WithFields(loglayer.Fields{"requestId": "abc"})

log.WithMetadata(loglayer.Metadata{"duration_ms": 120}).
    WithError(err).
    Error("request failed")
```

```json
{
  "msg": "request failed",
  "requestId": "abc",
  "duration_ms": 120,
  "err": { "message": "..." }
}
```
