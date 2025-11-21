package util

func ConvertStringishMap[M2 ~map[K2]V2, K1, K2 ~string, V1, V2 ~string | ~[]byte, M1 ~map[K1]V1](m1 M1) M2 {
	m2 := make(M2, len(m1))
	for k, v := range m1 {
		m2[K2(k)] = V2(v)
	}

	return m2
}
