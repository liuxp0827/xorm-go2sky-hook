# xorm-go2sky-hook

skywalking hook for xorm

## Example
```go
package main

import (
    "log"

    go2skyHook "github.com/liuxp0827/xorm-go2sky-hook"
    "github.com/SkyAPM/go2sky"
    "xorm.io/xorm"
)


func main() {
    db, err := xorm.NewEngine("sqlite3", "test.db")
    if err != nil {
        log.Fatal(err)
    }

    tracer, err := go2sky.NewTracer("127.0.0.1:11800")
    if err != nil {
        log.Fatal(err)
    }
    
    go2skyHook.WrapEngine(db, tracer)
    
    ...
}
```