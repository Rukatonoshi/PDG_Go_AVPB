package main

import "fmt"

func main() {
	x := 10
	y := 20
	n := 5
	for i := 1; i < n; i++ {
		if x > y {
			fmt.Printf("%d is greater than %d\n", x, y)
			y = x + n*i
		} else if x < y {
			fmt.Println("x is less than y")
			x = y + n*i
		} else {
			fmt.Println("x is equal to y")
			x++
		}
	}
}
