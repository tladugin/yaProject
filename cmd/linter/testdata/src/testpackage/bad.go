package main

import (
	"log"
	"os"
)

func badFunc() {
	panic("should be reported") // want "use of built-in panic function is forbidden"

	log.Fatal("should be reported") // want "call to log.Fatal forbidden outside main function"

	os.Exit(1) // want "call to os.Exit forbidden outside main function"
}

func anotherBadFunc() {
	log.Fatalf("formatted %s", "error") // want "call to log.Fatalf forbidden outside main function"

	log.Fatalln("fatal line") // want "call to log.Fatalln forbidden outside main function"
}
