@rem go mod init sqlschemakit
@rem pause
@rem go install golang.org/x/tools/cmd/goyacc@latest
@rem pause
go generate ./sqlschema
@pause
go build ./...
@pause