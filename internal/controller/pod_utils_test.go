// Copyright 2024 Universidad Carlos III de Madrid
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

// generate_test.go
package controller

import (
	"regexp"
	"testing"
)

// TestGenerate4byteChunk verifies that Generate4byteChunk produces a 4-character hexadecimal string.
func TestGenerate4byteChunk(t *testing.T) {
	// Regular expression to match exactly 4 hexadecimal characters
	re := regexp.MustCompile(`^[0-9a-fA-F]{4}$`)

	// Call Generate4byteChunk multiple times to check if output is always 4 characters
	for i := 0; i < 100; i++ {
		output := Generate4byteChunk()

		// Check if the output matches the 4-character hex pattern
		if !re.MatchString(output) {
			t.Errorf("Expected a 4-character hexadecimal string, but got: %s", output)
		}
	}
}
