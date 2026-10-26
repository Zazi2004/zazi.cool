// Command hashpw prints the bcrypt hash of the password read on standard input, to
// put in ADMIN_PASSWORD_HASH. Use `make hash-password`, which reads the password
// without echoing it.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"zazi.cool/home/internal/auth"
)

func main() {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr, "no password on standard input")
		os.Exit(1)
	}
	hash, err := auth.HashPassword(strings.TrimRight(line, "\r\n"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(hash)
	fmt.Fprintln(os.Stderr, "Dans .env, entourer le hash de guillemets simples : ADMIN_PASSWORD_HASH='…'")
}
