rem go generate 只要执行一次就会生成resource.syso，改变了json或图标的话，需要再次生成
rem go generate
go build -ldflags="-H windowsgui"