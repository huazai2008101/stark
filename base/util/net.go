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
	"net"
	"strings"
)

var localIPv4Str = "127.0.0.1"
var isLoadIpv4 bool

// LocalIPv4 获取本机的 IPv4 地址。
func LocalIPv4() string {
	if isLoadIpv4 {
		return localIPv4Str
	}
	conn, err := net.Dial("udp", "114.114.114.114:53")
	if err == nil {
		conn.Close()
		isLoadIpv4 = true
		localIPv4Str = strings.Split(conn.LocalAddr().String(), ":")[0]
		return localIPv4Str
	}
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		addrs, err := inter.Addrs()
		if err != nil {
			return localIPv4Str
		}
		for _, addr := range addrs {
			if addr.(*net.IPNet).IP.To4() != nil && addr.(*net.IPNet).IP.String() != "127.0.0.1" {
				if len(inter.Name) >= 1 && string(inter.Name[0]) == "e" {
					isLoadIpv4 = true
					localIPv4Str = addr.(*net.IPNet).IP.String()
					break
				}
			}
		}
	}

	return localIPv4Str
}
