genpb:
	protoc --go_out=./fixtures --go-grpc_out=./fixtures ./fixtures/*.proto

docker-up:
	docker compose -f scripts/setup/docker-compose.yml up -d
	./scripts/ci/wait_mysql_start.sh
	mysql -h 127.0.0.1 -P 3306 -u root -p'root' < scripts/setup/init.sql

docker-down:
	docker compose -f scripts/setup/docker-compose.yml down

docker-restart:
	make docker-down
	make docker-up
