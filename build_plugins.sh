#!/bin/bash

# source
go build --buildmode=plugin -o plugins/sources/mainflux/mainflux.so plugins/sources/mainflux/mainflux.go
cp plugins/sources/mainflux/mainflux.so ~/Programs/kuiper/plugins/sources/
cp plugins/sources/mainflux/mainflux.yaml ~/Programs/kuiper/etc/sources/

# sink
go build --buildmode=plugin -o plugins/sinks/mainflux/mainflux.so plugins/sinks/mainflux/mainflux.go
cp plugins/sinks/mainflux/mainflux.so ~/Programs/kuiper/plugins/sinks/
