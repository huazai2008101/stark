/*
 * Copyright 2012-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"bytes"
	"fmt"
	"runtime"
)

// PanicCond 封装触发 panic 的条件。
type PanicCond struct {
	fn func() interface{}
}

// When 满足给定条件时抛出一个 panic 。
func (p *PanicCond) When(isPanic bool) {
	if isPanic {
		panic(p.fn())
	}
}

// NewPanicCond PanicCond 的构造函数。
func NewPanicCond(fn func() interface{}) *PanicCond {
	return &PanicCond{fn}
}

// Panic 抛出一个异常值。
func Panic(err error) *PanicCond {
	return NewPanicCond(func() interface{} { return err })
}

// Panicf 抛出一段需要格式化的错误字符串。
func Panicf(format string, a ...interface{}) *PanicCond {
	return NewPanicCond(func() interface{} { return fmt.Errorf(format, a...) })
}

// 打印panic堆栈信息
func PanicStack() []byte {
	s := []byte("/src/runtime/panic.go")
	e := []byte("\ngoroutine ")
	line := []byte("\n")
	stack := make([]byte, 4<<10) //4KB
	length := runtime.Stack(stack, true)
	start := bytes.Index(stack, s)
	stack = stack[start:length]
	start = bytes.Index(stack, line) + 1
	stack = stack[start:]
	end := bytes.LastIndex(stack, line)
	if end != -1 {
		stack = stack[:end]
	}
	end = bytes.Index(stack, e)
	if end != -1 {
		stack = stack[:end]
	}
	stack = bytes.TrimRight(stack, "\n")
	return stack
}
