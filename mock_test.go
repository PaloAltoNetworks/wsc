// Copyright 2019 Aporeto Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wsc

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWSC_Mock(t *testing.T) {

	Convey("Given I have a MockWebsocket", t, func() {

		m := NewMockWebsocket(context.Background())

		Convey("When I set NextRead then read", func() {

			m.NextRead([]byte("hello"))

			out := <-m.Read()

			Convey("Then out should be correct", func() {
				So(string(out), ShouldEqual, "hello")
			})
		})

		Convey("When I write someting then get LastWrite", func() {

			m.Write([]byte("hello"))

			out := <-m.LastWrite()

			Convey("Then out should be correct", func() {
				So(string(out), ShouldEqual, "hello")
			})
		})

		Convey("When I send done then get LastDone", func() {

			m.NextDone(errors.New("boom"))

			out := <-m.Done()

			Convey("Then out should be correct", func() {
				So(out.Error(), ShouldEqual, "boom")
			})
		})

		Convey("When I call close", func() {

			m.Close(1)

			out := <-m.Done()

			Convey("Then out should be correct", func() {
				So(out.Error(), ShouldEqual, "1")
			})
		})
	})

}
