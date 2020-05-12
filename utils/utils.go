package utils

// Blow up if our `error` value was `nil`.
func Check(err error) {
	if err != nil {
		panic(err)
	}
}
