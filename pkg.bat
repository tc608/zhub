SET GOOS=linux
SET GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w" ./main.go
upx -9 zhub.sh
