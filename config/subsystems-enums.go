// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"reflect"

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
)

type StringValidator interface {
	Set(v string) error
	String() string
}

func init() {
	RegisterTypeDecodeHooks(enumDecodeHookFunc())
}

func enumDecodeHookFunc() mapstructure.DecodeHookFunc {
	model := reflect.TypeOf((*StringValidator)(nil)).Elem() // reflect.Type of StringValidator

	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// return if not from string or not to api.Version
		if f.Kind() != reflect.String {
			return data, nil
		}
		if !t.Implements(model) {
			return data, nil
		}

		// create a new value of the target type, convert to it StringEnumer,
		// parse the input into the value. Return error if parsing fails
		val := reflect.New(t.Elem()).Interface()
		if enumer, ok := val.(StringValidator); !ok {
			log.Warnf("(likely bug) subsystems-enum decode hook failed to cast %T to StringValidator interface; skipping", val)
			return data, nil
		} else {
			err := enumer.Set(data.(string))
			if err != nil {
				return data, err
			} else {
				return enumer, nil
			}
		}
	}
}
