package txapi

/*
Return a function that returns the next item from 'pool' every time. When 'pool' runs
out, keep returning the last item forever.

    backoff := getBackoff([]int{1, 2, 3})
    fmt.Println(backoff())
    // <<< 1
    fmt.Println(backoff())
    // <<< 2
    fmt.Println(backoff())
    // <<< 3
    fmt.Println(backoff())
    // <<< 3
    fmt.Println(backoff())
    // <<< 3
    // ...
*/
func getBackoff(pool []int) func() int {
	if pool == nil {
		pool = []int{1, 1, 1, 2, 3, 5, 8, 13}
	}
	i := -1
	return func() int {
		i++
		if i < len(pool) {
			return pool[i]
		} else {
			return pool[len(pool)-1]
		}
	}
}
