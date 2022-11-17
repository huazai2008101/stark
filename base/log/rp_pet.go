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

package log

import (
	"bytes"
	"fmt"

	"github.com/huazai2008101/stark/base/cast"
	"github.com/huazai2008101/stark/base/util"
	"github.com/maybgit/glog"
)

func init() {
	RegisterAppenderFactory("RpPetAppender", new(RpPetAppenderFactory))
}

type RpPetAppenderFactory struct{}

func (f *RpPetAppenderFactory) NewAppenderConfig() AppenderConfig {
	return new(RpPetAppenderConfig)
}

func (f *RpPetAppenderFactory) NewAppender(config AppenderConfig) (Appender, error) {
	return NewRpPetAppender(config.(*RpPetAppenderConfig)), nil
}

type RpPetAppenderConfig struct {
	Name string `xml:"name,attr"`
}

func (c *RpPetAppenderConfig) GetName() string {
	return c.Name
}

type RpPetAppender struct {
	config *RpPetAppenderConfig
}

func NewRpPetAppender(config *RpPetAppenderConfig) *RpPetAppender {
	return &RpPetAppender{config: config}
}

func (c *RpPetAppender) Append(msg *Message) {
	level := msg.Level()
	logFn := glog.Info
	if level >= ErrorLevel {
		logFn = glog.Error
	} else if level == WarnLevel {
		logFn = glog.Warning
	}
	var buf bytes.Buffer
	for _, a := range msg.Args() {
		buf.WriteString(cast.ToString(a))
	}
	fileLine := util.Contract(fmt.Sprintf("%s:%d", msg.File(), msg.Line()), 48)
	if msg.Context() != nil {
		logFn(msg.Context(), fmt.Sprintf("[%s] %s %s", fileLine, msg.tag, buf.String()))
	} else {
		logFn(fmt.Sprintf("[%s] %s %s", fileLine, msg.tag, buf.String()))
	}
}
