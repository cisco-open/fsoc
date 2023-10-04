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
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type Validator interface {
	ValidateAndSet(v any) error
	//String() string
}

func init() {
	RegisterTypeDecodeHooks(validatorDecodeHookFunc())
}

func validatorDecodeHookFunc() mapstructure.DecodeHookFunc {
	model := reflect.TypeOf((*Validator)(nil)).Elem() // reflect.Type of Validator

	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// chain to next hook if not converting to a validated value type
		if !t.Implements(model) {
			return data, nil
		}

		// create a new value of the target type & convert to it a Validator interface
		val, ok := reflect.New(t.Elem()).Interface().(Validator)
		if !ok {
			// TODO: consider whether to fail here rather than chaining it
			return data, fmt.Errorf("(likely bug) the type decode hook failed to cast %T to a Validator interface for subsystem config setting of type %q of package %q: ", val, t.Name(), t.PkgPath())
		}

		// parse the input into the value, return error with info if parsing fails
		err := val.ValidateAndSet(data)
		if err != nil {
			return data, err
		} else {
			return val, nil
		}
	}
}
