
@echo off

rem 获取当前日期和时间并格式化为 YYYY.MM.DD-HH.MM.SS
for /f "tokens=2 delims==" %%i in ('wmic os get localdatetime /value') do set datetime=%%i
set year=%datetime:~0,4%
set month=%datetime:~4,2%
set day=%datetime:~6,2%
set hour=%datetime:~8,2%
set minute=%datetime:~10,2%
set second=%datetime:~12,2%
set version=%year%.%month%.%day%-%hour%.%minute%.%second%

rem 删除历史编译文件
del zhub.sh zhub.exe zhub

rem Linux
set GOOS=linux
set GOARCH=amd64
go build -o zhub.sh -ldflags "-s -w -X 'zhub/internal/monitor.Version=%version%'"
upx -9 zhub.sh

rem Windows
set GOOS=windows
set GOARCH=amd64
go build -o zhub.exe -ldflags "-s -w -X 'zhub/internal/monitor.Version=%version%'"
upx -9 zhub.exe

rem Mac
set GOOS=darwin
set GOARCH=amd64
go build -o zhub -ldflags "-s -w -X 'zhub/internal/monitor.Version=%version%'"
upx -9 zhub

move /Y zhub.sh ./tmp/zhub/
move /Y zhub.exe ./tmp/zhub/
move /Y zhub ./tmp/zhub/
