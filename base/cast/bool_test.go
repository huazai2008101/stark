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

package cast_test

import (
	"strconv"
	"testing"

	"github.com/huazai2008101/stark/base/cast"
)

func BenchmarkToBool(b *testing.B) {
	// string/strconv-8    957624752 1.27 ns/op
	// string/stark-8  41272039  28.3 ns/op
	b.Run("string", func(b *testing.B) {
		v := "true"
		b.Run("strconv", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := strconv.ParseBool(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("stark", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := cast.ToBoolE(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
