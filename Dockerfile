FROM golang:latest

WORKDIR /go/src/github.com/rajeshreddyt/GrpcServerAdd

COPY . .

RUN go get -d -v ./...

RUN go install -v ./...

CMD ["GrpcServerAdd"]


# docker build -t mygrpcserver .
# docker run -p 50051:50051 mygrpcserver
# protoc --go_out=. path/to/your.proto

# protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative pkg/gopher/gopher.proto
# 
# run Hello.proto

# protoc --go_out=plugins=grpc:. --go_opt=paths=./ hello/Hello.proto

# protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./Hello.proto

# github.com/rajeshreddyt/GrpcServer
# protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative Hello/Hello.proto






# set path
# go get -u google.golang.org/protobuf
# go get -u google.golang.org/protobuf/protec-gen-go
# export PATH=$PATH:~/go/bin

# run the protoc

# protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative SUBTRACT/SUBTRACT.proto

# build image
# docker build -t rajeshreddyt/grpcserveradd-worker:latest .

# docker run -p 50051:50051 grpcserveradd

# push to docker hub

# docker tag grpcserveradd rajeshreddyt/grpcserveradd-worker:latest
# docker push rajeshreddyt/grpcserveradd-worker:latest

# docker tag rajeshreddyt/grpcserveradd-worker:latest localhost:5000/rajeshreddyt/grpcserveradd-worker:latest