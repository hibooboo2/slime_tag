
run:
	go mod tidy
	GOOS=windows go build -o slime_tag.exe
	./slime_tag.exe


.PHONY: spriteviewer
spriteviewer:
	go mod tidy
	GOOS=windows go build -o spriteviewer.exe cmd/spriteviewer/*.go
	./spriteviewer.exe $(file)
