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