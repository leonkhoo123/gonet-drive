package main
import (
	"fmt"
	"io/fs"
	"os"
)
func main() {
	info, _ := os.Stat("test_fs.go")
	entry := fs.FileInfoToDirEntry(info)
	fmt.Println(entry.Name())
}
