package assets

import (
	"embed"
	"fmt"
	"log"
	"path"
)

//go:embed resources
var Resources embed.FS

func printEmbededFSFiles(dir string) {
	entries, err := Resources.ReadDir(dir)
	if err != nil {
		log.Println(err) // Log and exit if there's an error
		return
	}

	for _, entry := range entries {
		fmt.Println(path.Join(dir, entry.Name()))
		if entry.IsDir() {
			// Recursively call the function for subdirectories
			printEmbededFSFiles(path.Join(dir, entry.Name()))
		}
	}
}
