package wsc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/smartystreets/goconvey/convey"
)

func TestWSC_ReadWrite(t *testing.T) {

	Convey("Given I have a webserver that works", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			h, err := Accept(ctx, ws, Config{})
			if err != nil {
				panic(err)
			}

			h.Write(<-h.Read())

			<-ctx.Done()
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, resp, err := Connect(
				ctx,
				strings.Replace(ts.URL, "http://", "ws://", 1),
				Config{},
			)

			Convey("Then err should be nil", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then resp should be correct", func() {
				So(resp, ShouldNotBeNil)
				So(resp.Status, ShouldEqual, "101 Switching Protocols")
			})

			Convey("When I listen for a message", func() {

				ws.Write([]byte("hello"))
				msg := <-ws.Read()

				Convey("Then msg should be correct", func() {
					So(string(msg), ShouldEqual, "hello")
				})

				Convey("When I close the connection", func() {

					doneErr := make(chan error)
					go func() {
						select {
						case e := <-ws.Done():
							doneErr <- e
						case <-ctx.Done():
							doneErr <- errors.New("test: no response in time")
						}
					}()

					ws.Close(0)

					Convey("Then doneErr should be nil", func() {
						So(<-doneErr, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestWSC_ConnectToServerWithHTTPError(t *testing.T) {

	Convey("Given I have a webserver that returns an http error", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope man", http.StatusForbidden)
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, resp, err := Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("Then ws should be nil", func() {
				So(ws, ShouldBeNil)
			})

			Convey("Then err should not be nil", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "websocket: bad handshake")
			})

			Convey("Then resp should be correct", func() {
				So(resp, ShouldNotBeNil)
				So(resp.Status, ShouldEqual, "403 Forbidden")
			})
		})
	})
}

func TestWSC_CannotConnect(t *testing.T) {

	Convey("Given I have a no webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		Convey("When I connect to the non existing server", func() {

			ws, resp, err := Connect(ctx, "ws://127.0.0.1:7745", Config{})

			Convey("Then ws should be nil", func() {
				So(ws, ShouldBeNil)
			})

			Convey("Then err should not be nil", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEndWith, "connection refused")
			})

			Convey("Then resp should be nil", func() {
				So(resp, ShouldBeNil)
			})
		})
	})
}

func TestWSC_GentleServerDisconnection(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			h, err := Accept(ctx, ws, Config{})
			if err != nil {
				panic(err)
			}

			h.Close(0)
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, _, _ := Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("When I wait for a message", func() {

				var err error
				select {
				case err = <-ws.Done():
				case <-ws.Read():
					panic("test: should not have received message")
				case <-ctx.Done():
					panic("test: no response in time")
				}

				Convey("Then err should be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "websocket: close 1001 (going away)")
				})
			})
		})
	})
}

func TestWSC_BrutalServerDisconnection(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}
			ws.Close() // nolint: errcheck
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, _, _ := Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("When I wait for a message", func() {

				var err error
				select {
				case err = <-ws.Done():
				case <-ws.Read():
					panic("test: should not have received message")
				case <-ctx.Done():
					panic("test: no response in time")
				}

				Convey("Then err should be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "websocket: close 1006 (abnormal closure): unexpected EOF")
				})
			})
		})
	})
}

func TestWSC_GentleClientDisconnection(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		rcvmsg := make(chan []byte)
		rcvdone := make(chan error)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			h, err := Accept(ctx, ws, Config{})
			if err != nil {
				panic(err)
			}

			select {
			case err = <-h.Done():
				rcvdone <- err
			case msg := <-h.Read():
				rcvmsg <- msg
			case <-ctx.Done():
				panic("test: no response in time")
			}

		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, _, _ := Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("When I gracefully stop the connection", func() {

				ws.Close(websocket.CloseInvalidFramePayloadData)

				var err error
				var msg []byte
				select {
				case err = <-rcvdone:
				case msg = <-rcvmsg:
				case <-time.After(1 * time.Second):
					panic("test: no response in time")
				}

				Convey("Then the err received by the client not be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "websocket: close 1007 (invalid payload data)")
				})

				Convey("Then no msg should be received by the client", func() {
					So(msg, ShouldBeNil)
				})
			})
		})
	})
}

func TestWSC_BrutalClientDisconnection(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		rcvmsg := make(chan []byte)
		rcvdone := make(chan error)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			h, err := Accept(ctx, ws, Config{})
			if err != nil {
				panic(err)
			}

			select {
			case err = <-h.Done():
				rcvdone <- err
			case msg := <-h.Read():
				rcvmsg <- msg
			case <-ctx.Done():
				panic("test: no response in time")
			}
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			w, _, _ := Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("When I gracefully stop the connection", func() {

				w.(*ws).conn.Close() // nolint: errcheck

				var err error
				var msg []byte
				select {
				case err = <-rcvdone:
				case msg = <-rcvmsg:
				case <-ctx.Done():
					panic("test: no response in time")
				}

				Convey("Then the err received by the server not be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "websocket: close 1006 (abnormal closure): unexpected EOF")
				})

				Convey("Then no msg should be received by the server", func() {
					So(msg, ShouldBeNil)
				})
			})
		})
	})
}

func TestWSC_ServerMissingPong(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			_, err = Accept(ctx, ws, Config{})
			if err != nil {
				panic(err)
			}

			<-ctx.Done()
		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			ws, _, _ := Connect(
				ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{
					PongWait:   1 * time.Nanosecond, // we wait for nothing
					PingPeriod: 50 * time.Millisecond,
				})

			Convey("When I wait for a message", func() {

				<-time.After(300 * time.Millisecond)

				var err error
				var msg []byte
				select {
				case err = <-ws.Done():
				case msg = <-ws.Read():
				case <-ctx.Done():
					panic("test: no response in time")
				}

				Convey("Then the err received by the client not be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEndWith, "i/o timeout")
				})

				Convey("Then no msg should be received by the client", func() {
					So(msg, ShouldBeNil)
				})
			})
		})
	})
}

func TestWSC_ClientMissingPong(t *testing.T) {

	Convey("Given I have a webserver", t, func() {

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		rcvmsg := make(chan []byte)
		rcvdone := make(chan error)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ws, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				panic(err)
			}

			h, err := Accept(ctx, ws, Config{
				PongWait:   1 * time.Millisecond,
				PingPeriod: 50 * time.Millisecond,
			})
			if err != nil {
				panic(err)
			}

			select {
			case err = <-h.Done():
				rcvdone <- err
			case msg := <-h.Read():
				rcvmsg <- msg
			case <-ctx.Done():
				panic("test: no response in time")
			}

		}))
		defer ts.Close()

		Convey("When I connect to the webserver", func() {

			_, _, _ = Connect(ctx, strings.Replace(ts.URL, "http://", "ws://", 1), Config{})

			Convey("When I wait for a message", func() {

				<-time.After(300 * time.Millisecond)

				var err error
				var msg []byte
				select {
				case err = <-rcvdone:
				case msg = <-rcvmsg:
				case <-ctx.Done():
					panic("test: no response in time")
				}

				Convey("Then the err received by the server not be nil", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEndWith, "i/o timeout")
				})

				Convey("Then no msg should be received by the server", func() {
					So(msg, ShouldBeNil)
				})
			})
		})
	})
}
