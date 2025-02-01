
run:
	go mod tidy
	GOOS=windows go build -o slime_tag.exe
	./slime_tag.exe

