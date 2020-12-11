// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpcdb

import (
	"log"
	"net"
	"testing"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/liraxapp/avalanchego/database"
	"github.com/liraxapp/avalanchego/database/memdb"
	"github.com/liraxapp/avalanchego/database/rpcdb/rpcdbproto"
)

const (
	bufSize = 1 << 20
)

func TestInterface(t *testing.T) {
	for _, test := range database.Tests {
		listener := bufconn.Listen(bufSize)
		server := grpc.NewServer()
		rpcdbproto.RegisterDatabaseServer(server, NewServer(memdb.New()))
		go func() {
			if err := server.Serve(listener); err != nil {
				log.Fatalf("Server exited with error: %v", err)
			}
		}()

		dialer := grpc.WithContextDialer(
			func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			})

		ctx := context.Background()
		conn, err := grpc.DialContext(ctx, "", dialer, grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial: %s", err)
		}

		db := NewClient(rpcdbproto.NewDatabaseClient(conn))
		test(t, db)
		conn.Close()
	}
}
