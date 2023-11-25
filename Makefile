.PHONY: *

gogo: stop-services build truncate-logs start-services

build:
	cd webapp/go && make build

stop-services:
	sudo systemctl stop nginx
	sudo systemctl stop isupipe-go.service
	sudo systemctl stop pdns.service
	sudo systemctl stop mysql

start-services:
	sudo systemctl start mysql
	sleep 2
	sudo systemctl start isupipe-go.service
	sleep 1
	sudo systemctl start pdns.service
	sleep 1
	sudo systemctl start nginx

truncate-logs:
	sudo truncate --size 0 /var/log/nginx/access.log
	sudo truncate --size 0 /var/log/nginx/error.log
	sudo truncate --size 0 /var/log/mysql/mysql-slow.log
	sudo chmod 777 /var/log/mysql/mysql-slow.log
	sudo journalctl --vacuum-size=1K

kataribe:
	cd ~/ && sudo cat /var/log/nginx/access.log | ./kataribe

pprof: TIME=60
pprof: PROF_FILE=~/pprof.samples.$(shell TZ=Asia/Tokyo date +"%H%M").$(shell git rev-parse HEAD | cut -c 1-8).pb.gz
pprof:
	curl -sSf "http://localhost:6060/debug/fgprof?seconds=$(TIME)" > $(PROF_FILE)
	go tool pprof $(PROF_FILE)
