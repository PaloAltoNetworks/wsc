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
