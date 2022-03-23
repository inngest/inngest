package faker

import (
	"fmt"
	"strconv"
	"strings"
)

func extractFloat64FromTagArgs(args []string) (interface{}, error) {
	bytes := 64
	var floatValues []float64
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseFloat(k, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, j)
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractFloat32FromTagArgs(args []string) (interface{}, error) {
	bytes := 32
	var floatValues []float32
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseFloat(k, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, float32(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractInt64FromTagArgs(args []string) (interface{}, error) {
	bytes := 64
	var floatValues []int64
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseInt(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, j)
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractInt32FromTagArgs(args []string) (interface{}, error) {
	bytes := 32
	var floatValues []int32
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseInt(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, int32(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractInt16FromTagArgs(args []string) (interface{}, error) {
	bytes := 16
	var floatValues []int16
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseInt(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, int16(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractInt8FromTagArgs(args []string) (interface{}, error) {
	bytes := 8
	var floatValues []int8
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseInt(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, int8(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractIntFromTagArgs(args []string) (interface{}, error) {
	bytes := 0
	var floatValues []int
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseInt(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, int(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractUint64FromTagArgs(args []string) (interface{}, error) {
	bytes := 64
	var floatValues []uint64
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseUint(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, j)
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractUint32FromTagArgs(args []string) (interface{}, error) {
	bytes := 32
	var floatValues []uint32
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseUint(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, uint32(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractUint16FromTagArgs(args []string) (interface{}, error) {
	bytes := 16
	var floatValues []uint16
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseUint(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, uint16(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractUint8FromTagArgs(args []string) (interface{}, error) {
	bytes := 8
	var floatValues []uint8
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseUint(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, uint8(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}

func extractUintFromTagArgs(args []string) (interface{}, error) {
	bytes := 0
	var floatValues []uint
	for _, i := range args {
		k := strings.TrimSpace(i)
		j, err := strconv.ParseUint(k, 10, bytes)
		if err != nil {
			return nil, fmt.Errorf(ErrUnsupportedTagArguments)
		}
		floatValues = append(floatValues, uint(j))
	}
	toRet := floatValues[rand.Intn(len(floatValues))]
	return toRet, nil
}
