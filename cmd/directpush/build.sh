set -a; source ../../.env; set +a;
go build -o order-api
./order-api