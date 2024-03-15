package slice

func IndexOf[T any](slice []T, predicate func(T) bool) int {
	for i, val := range slice {
		if predicate(val) {
			return i
		}
	}
	return -1
}

func Remove[T any](slice []T, predicate func(T) bool) []T {
	i := IndexOf(slice, predicate)
	if i == -1 {
		return slice
	}
	return append(slice[:i], slice[i+1:]...)
}

func Contains[T comparable](slice []T, val T) bool {
	for _, element := range slice {
		if element == val {
			return true
		}
	}
	return false
}

func GetLeftDifference[T comparable](left []T, right []T) (leftDifference []T) {
	for _, val := range left {
		if !Contains[T](right, val) {
			leftDifference = append(leftDifference, val)
		}
	}
	return leftDifference
}
