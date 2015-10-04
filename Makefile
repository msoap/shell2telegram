run:
	go run $$(ls *.go | grep -v _test.go) $${TB_ROOT:+-root-users=$$TB_ROOT} -add-exit /date date /:plain_text 'cat -n'

build:
	go build

update-from-github:
	go get -u github.com/msoap/shell2telegram
