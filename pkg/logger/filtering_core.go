// Copyright 2017 Quentin Machu & eco authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"go.uber.org/zap/zapcore"
)

// FilterFunc is used to check whether to filter the given field out.
type FilterFunc func(zapcore.Entry) bool

type filteringCore struct {
	zapcore.Core
	filter FilterFunc
}

// NewFilteringCore returns a core that uses the given filter function
// to filter event based on fields before passing them to the core being wrapped.
//
// FilterFunc should return false to skip the log entry, true to write it.
func NewFilteringCore(next zapcore.Core, filter FilterFunc) zapcore.Core {
	return &filteringCore{next, filter}
}

func (core *filteringCore) With(fields []zapcore.Field) zapcore.Core {
	return core.Core.With(fields)
}

func (core *filteringCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !core.filter(entry) {
		return checkedEntry
	}
	return core.Check(entry, checkedEntry)
}

func (core *filteringCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return core.Core.Write(entry, fields)
}
