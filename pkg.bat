SET GOOS=linux
SET GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w"
upx -9 zhub.sh

rem scp zhub.sh  dev:/opt/zhub
