
@echo off

rem 删除历史编译文件
del zhub.sh zhub.exe zhub

rem Linux
set GOOS=linux
set GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w"
upx -9 zhub.sh

rem Windows
set GOOS=windows
set GOARCH=amd64
go build -o zhub.exe -ldflags "-s -w"
upx -9 zhub.exe

rem Mac
set GOOS=darwin
set GOARCH=amd64
go build -o zhub -ldflags "-s -w"
upx -9 zhub

move /Y zhub.sh ./tmp/zhub/
move /Y zhub.exe ./tmp/zhub/
move /Y zhub ./tmp/zhub/
