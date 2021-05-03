SET GOOS=linux
SET GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w" ./app.go
upx -9 zhub.sh

scp zhub.sh  pro:/opt/zhub
scp zhub.sh  dev:/opt/zhub
scp zhub.sh  qc:/opt/zhub
scp zhub.sh  my:/opt/zhub
del zhub.sh
