# zg
A simple golang log library.


This project bases on [uber/zap](https://github.com/uber-go/zap), a wonderful log library for golang.


### Trace logs
Logs should be tracable. Logs in one request or one flow should be grouped.

Example:
```go
package main

import (
    "context"

    "github.com/xslook/zg"
)

func doThings(ctx context.Context, num int) {
    zg.In(ctx).With(zg.I("num", num)).Info("do things")
    // other logic codes
}

func main() {
    ctx := context.Background()
    ctx = zg.Trace(ctx) // Trace this context

    doThings(ctx, 1)
    doThings(ctx, 2)
}
```
It will log somethings like below:
```
```


### Rotate Log File
Why rotate?

- If you keep write logs into a file, it may grow too large.
- You may want split your log files daily.

But **zg** DO NOT provide a way rotate log file itself, instead provide a method **Reload**
to be notified after rotate finished.

Example:
```go
package main

import (
    "github.com/xslook/zg"
)

func main() {
    zg.Reload()
}
```



### LICENSE
MIT LICENSE

