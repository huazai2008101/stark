# atomic

封装标准库 atomic 包的操作函数。

## Install

```
go get github.com/huazai2008101/stark/base@v1.1.0-rc2
```

## Import

```
import "github.com/huazai2008101/stark/base/atomic"
```

## Example

```
var i64 atomic.Int64
i64.Add(1)
i64.Store(2)
_ = i64.Load()
```
