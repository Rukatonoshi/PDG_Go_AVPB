package main

func complexFunction() int {
	a := 0
	b := 1
	c := 3
	n := 4
	result := 0
	sum := 0

	for i := 0; i < n; i++ {
		if i % 2 {
			a += i
		} else {
			b += i
		}

		for j := 0; j < i; j++ {
			if j % 3 {
				c += j
			} else {
				sum += j
			}
		}

		if a > b {
			continue
		} else if b > c {
			break
		}
	}
	if sum > 10 {
		result = a + b
		return result
	} else {
		return c
	}
}
