# cast

提供了很多类型之间相互转换的函数。

## Install

```
go get github.com/huazai2008101/stark/base@v1.1.0-rc2
```

## Import

```
import "github.com/huazai2008101/stark/base/cast"
```

## Example

```
fmt.Println(cast.ToInt(10))   // 10
fmt.Println(cast.ToInt(10.0)) // 10
fmt.Println(cast.ToInt("10")) // 10
fmt.Println(cast.ToInt(true)) // 1
```
