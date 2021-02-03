SET GOOS=linux
SET GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w" ./app.go
upx -9 zhub.sh

scp zhub.sh  xhost:/opt/zhub
scp zhub.sh  zhost:/opt/zhub
scp zhub.sh  qhost:/opt/zhub
scp zhub.sh  nhost:/opt/zhub
del zhub.sh
