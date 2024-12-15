build:
	go build -o ./dist/shutterpilot .

test:
	go test

clear_tmp:
	rm -r tmp*
