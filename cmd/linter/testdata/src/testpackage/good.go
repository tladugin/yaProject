package main

import (
	"log"
	"os"
)

func main() {
	// Эти вызовы разрешены, так как находятся в main пакете и main функции
	log.Fatal("This is allowed")
	os.Exit(1)
	log.Fatalf("formatted %s", "error")
	log.Fatalln("fatal line")
}

func regularFunction() {
	// Этот panic все равно будет обнаружен
	panic("still forbidden") // want "use of built-in panic function is forbidden"
}
