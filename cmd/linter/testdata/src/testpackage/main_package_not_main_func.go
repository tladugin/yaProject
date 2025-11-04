package main

import (
	"log"
	"os"
)

func notMainFunction() {
	// Эти вызовы запрещены, так как не в main функции
	log.Fatal("should be reported") // want "call to log.Fatal forbidden outside main function"
	os.Exit(1)                      // want "call to os.Exit forbidden outside main function"
}

func anotherFunction() {
	panic("should be reported") // want "use of built-in panic function is forbidden"
}
