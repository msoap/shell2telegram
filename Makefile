run:
	go run $$(ls *.go | grep -v _test.go) $${TB_ROOT:+-root-users=$$TB_ROOT} \
        -add-exit \
        -log-commands \
        -persistent_users \
        /date:desc="Get current date" date \
        /:plain_text:desc="Numbers via cat -n" 'cat -n' \
        /alarm:vars=SLEEP,MSG 'sleep $$SLEEP; echo $$MSG'

build:
	go build

test:
	go test

update-from-github:
	go get -u github.com/msoap/shell2telegram
