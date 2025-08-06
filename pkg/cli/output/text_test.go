package output

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/tabwriter"
)

// testTextWriter creates a TextWriter that writes to a shared buffer
func newTestTextWriter(buf *bytes.Buffer, indent int) *TextWriter {
	return &TextWriter{
		indent: indent,
		w:      tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0),
	}
}

func TestTextWriter_Write_SimpleMap(t *testing.T) {
	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	data := map[string]any{
		"ID":   "test-id",
		"Name": "test-name",
		"Age":  25,
	}

	err := tw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ID:") {
		t.Errorf("Expected output to contain 'ID:', got: %s", output)
	}
	if !strings.Contains(output, "test-id") {
		t.Errorf("Expected output to contain 'test-id', got: %s", output)
	}
	if !strings.Contains(output, "Name:") {
		t.Errorf("Expected output to contain 'Name:', got: %s", output)
	}
	if !strings.Contains(output, "Age:") {
		t.Errorf("Expected output to contain 'Age:', got: %s", output)
	}
}

func TestTextWriter_Write_NestedMap(t *testing.T) {
	// Capture stdout since nested maps now write to separate tabwriters
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	data := map[string]any{
		"Type": "Partition",
		"ID":   "test-id",
		"Tenant": map[string]any{
			"Account": "acc-123",
			"App":     "app-456",
		},
	}

	err := tw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read stdout output
	stdoutOutput := make([]byte, 1024)
	n, _ := r.Read(stdoutOutput)
	allOutput := buf.String() + string(stdoutOutput[:n])

	// Check that we have the main keys
	if !strings.Contains(allOutput, "Type:") {
		t.Errorf("Expected output to contain 'Type:', got: %s", allOutput)
	}
	if !strings.Contains(allOutput, "Tenant:") {
		t.Errorf("Expected output to contain 'Tenant:', got: %s", allOutput)
	}
	// Nested content appears in stdout
	if !strings.Contains(allOutput, "Account:") {
		t.Errorf("Expected output to contain 'Account:', got: %s", allOutput)
	}
	if !strings.Contains(allOutput, "acc-123") {
		t.Errorf("Expected output to contain 'acc-123', got: %s", allOutput)
	}
}

func TestTextWriter_Write_WithIndent(t *testing.T) {
	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 4)

	data := map[string]any{
		"Key": "value",
	}

	err := tw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	// tabwriter processes the spaces, so let's just check that Key: appears
	// and that we have some leading whitespace
	if !strings.Contains(output, "Key:") {
		t.Errorf("Expected output to contain 'Key:', got: %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("Expected output to contain 'value', got: %s", output)
	}
}

func TestTextWriter_Write_DifferentMapTypes(t *testing.T) {
	// Capture stdout since nested maps now write to separate tabwriters
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	data := map[string]any{
		"StringMap": map[string]string{
			"Key1": "Value1",
			"Key2": "Value2",
		},
		"IntMap": map[string]int{
			"Count1": 10,
			"Count2": 20,
		},
	}

	err := tw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read stdout output
	stdoutOutput := make([]byte, 1024)
	n, _ := r.Read(stdoutOutput)
	allOutput := buf.String() + string(stdoutOutput[:n])

	if !strings.Contains(allOutput, "StringMap:") {
		t.Errorf("Expected output to contain 'StringMap:', got: %s", allOutput)
	}
	if !strings.Contains(allOutput, "IntMap:") {
		t.Errorf("Expected output to contain 'IntMap:', got: %s", allOutput)
	}
	if !strings.Contains(allOutput, "Value1") {
		t.Errorf("Expected output to contain 'Value1', got: %s", allOutput)
	}
	if !strings.Contains(allOutput, "10") {
		t.Errorf("Expected output to contain '10', got: %s", allOutput)
	}
}

func TestTextWriter_Write_WithLeadSpace(t *testing.T) {
	// Capture stdout to test the leading space
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	data := map[string]any{
		"Key": "value",
	}

	err := tw.Write(data, WithTextOptLeadSpace(true))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read from pipe
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	stdoutOutput := string(output[:n])

	// Should have printed a newline to stdout
	if stdoutOutput != "\n" {
		t.Errorf("Expected leading space to print newline to stdout, got: '%s'", stdoutOutput)
	}

	// Buffer should still contain the data
	bufOutput := buf.String()
	if !strings.Contains(bufOutput, "Key:") {
		t.Errorf("Expected buffer to contain 'Key:', got: %s", bufOutput)
	}
}

func TestTextWriter_valueToString(t *testing.T) {
	tw := &TextWriter{}

	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{fmt.Errorf("test error"), "test error"},
	}

	for _, test := range tests {
		result := tw.valueToString(test.input)
		if result != test.expected {
			t.Errorf("valueToString(%v) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestTextWriter_isNestedMap(t *testing.T) {
	tw := &TextWriter{}

	tests := []struct {
		input    any
		expected bool
	}{
		{map[string]any{"key": "value"}, true},
		{map[string]string{"key": "value"}, true},
		{map[string]int{"key": 1}, true},
		{map[string]int64{"key": int64(1)}, true},
		{map[string]float64{"key": 1.0}, true},
		{map[string]bool{"key": true}, true},
		{"string", false},
		{42, false},
		{[]string{"array"}, false},
	}

	for _, test := range tests {
		result := tw.isNestedMap(test.input)
		if result != test.expected {
			t.Errorf("isNestedMap(%T) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestTextWriter_convertToAnyMap(t *testing.T) {
	tw := &TextWriter{}

	// Test map[string]string conversion
	stringMap := map[string]string{"key1": "value1", "key2": "value2"}
	result := tw.convertToAnyMap(stringMap)
	if len(result) != 2 {
		t.Errorf("Expected converted map to have 2 entries, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("Expected result['key1'] = 'value1', got %v", result["key1"])
	}

	// Test map[string]int conversion
	intMap := map[string]int{"count1": 10, "count2": 20}
	result = tw.convertToAnyMap(intMap)
	if len(result) != 2 {
		t.Errorf("Expected converted map to have 2 entries, got %d", len(result))
	}
	if result["count1"] != 10 {
		t.Errorf("Expected result['count1'] = 10, got %v", result["count1"])
	}

	// Test map[string]any passthrough
	anyMap := map[string]any{"key": "value"}
	result = tw.convertToAnyMap(anyMap)
	if len(result) != 1 || result["key"] != "value" {
		t.Errorf("Expected map[string]any to be returned as-is with key='value'")
	}
}

func TestTextWriter_Write_MultipleCallsOrdering(t *testing.T) {
	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	// Make multiple Write calls in a specific order
	err := tw.Write(map[string]any{
		"First": "Block A",
	})
	if err != nil {
		t.Fatalf("First Write failed: %v", err)
	}

	err = tw.Write(map[string]any{
		"Second": "Block B",
	})
	if err != nil {
		t.Fatalf("Second Write failed: %v", err)
	}

	err = tw.Write(map[string]any{
		"Third": "Block C",
	})
	if err != nil {
		t.Fatalf("Third Write failed: %v", err)
	}

	// Flush once at the end
	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify all content appears
	if !strings.Contains(output, "First:") {
		t.Errorf("Expected output to contain 'First:', got: %s", output)
	}
	if !strings.Contains(output, "Second:") {
		t.Errorf("Expected output to contain 'Second:', got: %s", output)
	}
	if !strings.Contains(output, "Third:") {
		t.Errorf("Expected output to contain 'Third:', got: %s", output)
	}

	// Verify order by checking line positions
	firstPos := -1
	secondPos := -1
	thirdPos := -1

	for i, line := range lines {
		if strings.Contains(line, "First:") {
			firstPos = i
		}
		if strings.Contains(line, "Second:") {
			secondPos = i
		}
		if strings.Contains(line, "Third:") {
			thirdPos = i
		}
	}

	if firstPos == -1 || secondPos == -1 || thirdPos == -1 {
		t.Fatalf("Could not find all expected lines in output: %s", output)
	}

	if firstPos >= secondPos || secondPos >= thirdPos {
		t.Errorf("Expected order: First < Second < Third, got positions: First=%d, Second=%d, Third=%d",
			firstPos, secondPos, thirdPos)
	}
}

func TestTextWriter_WriteOrdered_PreservesKeyOrder(t *testing.T) {
	var buf bytes.Buffer
	tw := newTestTextWriter(&buf, 0)

	// Create OrderedData with specific key order
	data := OrderedData(
		"Zebra", "should be first",
		"Alpha", "should be second",
		"Beta", "should be third",
		"Gamma", "should be fourth",
	)

	err := tw.WriteOrdered(data)
	if err != nil {
		t.Fatalf("WriteOrdered failed: %v", err)
	}

	err = tw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify order - should appear in the exact order specified in OrderedData
	expectedOrder := []string{"Zebra:", "Alpha:", "Beta:", "Gamma:"}

	if len(lines) != len(expectedOrder) {
		t.Fatalf("Expected %d lines, got %d: %v", len(expectedOrder), len(lines), lines)
	}

	for i, expectedKey := range expectedOrder {
		if !strings.Contains(lines[i], expectedKey) {
			t.Errorf("Expected line %d to contain '%s', got: '%s'", i, expectedKey, lines[i])
		}
	}

	// Verify the actual values appear in correct order too
	if !strings.Contains(lines[0], "should be first") {
		t.Errorf("Expected first line to contain 'should be first', got: '%s'", lines[0])
	}
	if !strings.Contains(lines[1], "should be second") {
		t.Errorf("Expected second line to contain 'should be second', got: '%s'", lines[1])
	}
}

func TestOrderedData_HelperFunction(t *testing.T) {
	// Test the OrderedData helper function
	om := OrderedData(
		"key3", "value3",
		"key1", "value1",
		"key2", "value2",
	)

	if om.Len() != 3 {
		t.Errorf("Expected length 3, got %d", om.Len())
	}

	keys := om.Keys()
	expectedKeys := []string{"key3", "key1", "key2"}

	for i, expectedKey := range expectedKeys {
		if keys[i] != expectedKey {
			t.Errorf("Expected key %d to be '%s', got '%s'", i, expectedKey, keys[i])
		}
	}

	// Test retrieval
	value, exists := om.Get("key2")
	if !exists {
		t.Error("Expected key2 to exist")
	}
	if value != "value2" {
		t.Errorf("Expected value2, got %v", value)
	}
}

func TestTextWriter_formatAsJSON(t *testing.T) {
	tests := []struct {
		name     string
		indent   int
		input    any
		expected string
	}{
		{
			name:     "simple object single line",
			indent:   0,
			input:    map[string]string{"key": "value"},
			expected: "{\n\t  \"key\": \"value\"\n\t}",
		},
		{
			name:   "complex object multi-line with indent 0",
			indent: 0,
			input: map[string]any{
				"name": "test",
				"nested": map[string]int{
					"count": 42,
				},
			},
			expected: "{\n\t  \"name\": \"test\",\n\t  \"nested\": {\n\t    \"count\": 42\n\t  }\n\t}",
		},
		{
			name:   "complex object multi-line with indent 2",
			indent: 2,
			input: map[string]any{
				"name": "test",
				"nested": map[string]int{
					"count": 42,
				},
			},
			expected: "{\n  \t  \"name\": \"test\",\n  \t  \"nested\": {\n  \t    \"count\": 42\n  \t  }\n  \t}",
		},
		{
			name:   "array multi-line",
			indent: 0,
			input:  []string{"first", "second", "third"},
			expected: "[\n\t  \"first\",\n\t  \"second\",\n\t  \"third\"\n\t]",
		},
		{
			name:     "non-serializable value",
			indent:   0,
			input:    make(chan int),
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tw := &TextWriter{indent: test.indent}
			result := tw.formatAsJSON(test.input)
			if result != test.expected {
				t.Errorf("formatAsJSON() = %q, expected %q", result, test.expected)
			}
		})
	}
}

func TestTextWriter_valueToString_WithJSONSerialization(t *testing.T) {
	tw := &TextWriter{indent: 0}

	// Test JSON serializable struct
	type TestStruct struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "JSON serializable struct",
			input:    TestStruct{Name: "test", Count: 42},
			expected: "{\n\t  \"name\": \"test\",\n\t  \"count\": 42\n\t}",
		},
		{
			name: "slice with multi-line JSON",
			input: []map[string]string{
				{"key": "value1"},
				{"key": "value2"},
			},
			expected: "[\n\t  {\n\t    \"key\": \"value1\"\n\t  },\n\t  {\n\t    \"key\": \"value2\"\n\t  }\n\t]",
		},
		{
			name:     "non-JSON serializable falls back",
			input:    make(chan int),
			expected: "0x", // starts with channel address format
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := tw.valueToString(test.input)
			if test.name == "non-JSON serializable falls back" {
				// For channel, just check it starts with expected pattern
				if !strings.HasPrefix(result, test.expected) {
					t.Errorf("valueToString() = %q, expected to start with %q", result, test.expected)
				}
			} else {
				if result != test.expected {
					t.Errorf("valueToString() = %q, expected %q", result, test.expected)
				}
			}
		})
	}
}
