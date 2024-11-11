package main

import "fmt"

func main() {
	x := 10
	y := 20
	z := x + y
	if z >= 10 {
		z = z + x
		fmt.Print(z)
		return
	} else {
		fmt.Print(x)
	}
	z += 25
	fmt.Print(x)
}
