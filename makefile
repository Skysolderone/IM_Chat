run-user:
	./bin/user


run-gateway:
	./bin/gateway

build-user:
	go build -o bin/user user/user.go


run-client:
	go run gateway/client/client.go


build-gateway:
	go build -o bin/gateway gateway/gateway.go

scp-gateway:
	go build -ldflags "-s -w" -o bin/socket_gateway gateway/gateway.go
	sshpass -p "G8jQyT6hZ4wFb3N9mR1pK5cL" scp bin/socket_gateway ec_user@52.201.237.21:~/socket_gateway
	sshpass -p "G8jQyT6hZ4wFb3N9mR1pK5cL" ssh ec_user@52.201.237.21 

