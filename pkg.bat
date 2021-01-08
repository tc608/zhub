SET GOOS=linux
SET GOARCH=amd64
go build -o zdb.sh -ldflags "-s -w" ./main.go
upx -9 zdb.sh
