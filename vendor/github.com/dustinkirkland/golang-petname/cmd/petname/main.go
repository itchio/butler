/*
  petname: binary for generating human-readable, random names
           for objects (e.g. hostnames, containers, blobs)

  Copyright 2014 Dustin Kirkland <dustin.kirkland@gmail.com>

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
	"github.com/dustinkirkland/golang-petname"
)

var (
	words = flag.Int("words", 2, "The number of words in the pet name")
	separator = flag.String("separator", "-", "The separator between words in the pet name")
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	fmt.Println(petname.Generate(*words, *separator))
}
