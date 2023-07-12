// Copyright 2022 Cisco Systems, Inc.
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

package editor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func Run(in io.Reader) (edited []byte, err error) {
	envs := []string{"EDITOR", "VISUAL"}
	editor := NewDefaultEditor(envs)

	// Copy original
	inCopy := bytes.Buffer{}
	in = io.TeeReader(in, &inCopy)
	original, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0]))
	suffix := uuid.New().String()

	edited, file, err := editor.LaunchTempFile(prefix, suffix, &inCopy)
	if err != nil {
		return nil, err
	}

	// Cancel edit if content has not changed
	if bytes.Equal(original, edited) {
		os.Remove(file)
		return nil, fmt.Errorf("edit cancelled, no changes made")
	}

	// Check that file is not empty
	if len(edited) == 0 {
		os.Remove(file)
		return nil, fmt.Errorf("edited file is empty")
	}

	return edited, nil
}
