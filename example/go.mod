module heegrpc/example

go 1.13

replace github.com/heegspace/heegrpc => /home/hai/github/go/src/heegrpc

replace github.com/heegspace/heegrpc/rpc => /home/hai/github/go/src/heegrpc/rpc

replace github.com/heegspace/heegrpc/utils => /home/hai/github/go/src/heegrpc/utils

replace github.com/heegspace/heegrpc/registry => /home/hai/github/go/src/heegrpc/registry

require (
	github.com/heegspace/heegproto v0.0.4
	github.com/heegspace/heegrpc v0.0.0-00010101000000-000000000000 // indirect
	github.com/heegspace/heegrpc/registry v0.0.0-00010101000000-000000000000 // indirect
	github.com/heegspace/heegrpc/rpc v0.0.0-00010101000000-000000000000 // indirect
	github.com/heegspace/heegrpc/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/heegspace/thrift v0.0.1
)
