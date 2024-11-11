package main

func main() {
	testString = "Some string"
	fib(n)
}

func fib(int n) (c int) {
	a = 0
	b = 1

	if n < 2 {
		return n
	}

	for i := 1; i < n; i++ {
		c = a + b
		a = b
		if c == b {
			continue
		}
		b = c
	}
	return c
}

//Разделение блоков по данным +-
//Разделение типов связей (пунктир \ жирным) +
//Подсветка условий (For) зеленым или красным при люб ветв +
