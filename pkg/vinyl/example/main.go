package main

import (
	fmt "fmt"
	"log"

	"github.com/embly/vinyl/vinyl-go"
)

//go:generate protoc -I . ./tables.proto --go_out=plugins=grpc:.

func main() {
	db, err := vinyl.Connect("vinyl://what:ever@localhost:8090/foo",
		vinyl.Metadata{Descriptor: []byte{0xa, 0x89, 0x1, 0xa, 0xa, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x4, 0x64, 0x61, 0x74, 0x61, 0x22, 0x6d, 0xa, 0x4, 0x55, 0x73, 0x65, 0x72, 0x12, 0xe, 0xa, 0x2, 0x69, 0x64, 0x18, 0x1, 0x20, 0x1, 0x28, 0x3, 0x52, 0x2, 0x69, 0x64, 0x12, 0x1a, 0xa, 0x8, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x2, 0x20, 0x1, 0x28, 0x9, 0x52, 0x8, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0xa, 0x5, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x3, 0x20, 0x1, 0x28, 0x9, 0x52, 0x5, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x23, 0xa, 0xd, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x4, 0x20, 0x1, 0x28, 0x9, 0x52, 0xc, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x48, 0x61, 0x73, 0x68, 0x62, 0x6, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33},
			Records: []vinyl.Record{
				vinyl.Record{
					Name:       "User",
					PrimaryKey: "id",
					Indexes: []vinyl.Index{
						vinyl.Index{
							Field: "email", Unique: true,
						}, vinyl.Index{Field: "username", Unique: true}}}}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(db)
}