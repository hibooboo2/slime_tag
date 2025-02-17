
run:
	go mod tidy
	GOOS=windows go build -o slime_tag.exe
	./slime_tag.exe


.PHONY: spriteviewer
spriteviewer:
	go mod tidy
	GOOS=windows go build -o spriteviewer.exe cmd/spriteviewer/*.go
	./spriteviewer.exe $(file)


.PHONY: mapmaker
mapmaker:
	cd cmd/mapmaker && python3 -m http.server 8000

.PHONY: mapconstraintViewer
mapconstraintViewer:
	go mod tidy
	GOOS=windows go build -o mapconstraintViewer.exe cmd/mapconstraintViewer/*.go
	./mapconstraintViewer.exe $(file)
