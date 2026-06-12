package goldie

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"text/template"
)

// Assert compares the actual data received with the expected data in the
// golden files. If the update flag is set, it will also update the golden
// file.
//
// `name` refers to the name of the test, and it should typically be unique
// within the package. Also, it should be a valid file name (so keeping to
// `a-z0-9\-\_` is a good idea).
func (g *Goldie) Assert(t testing.TB, name string, actualData []byte) {
	t.Helper()
	if *update {
		err := g.Update(t, name, actualData)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	err := g.compare(t, name, actualData)
	if err != nil {
		{
			var e *errFixtureNotFound
			if errors.As(err, &e) {
				t.Error(err)
				t.FailNow()
				return
			}
		}

		{
			var e *errFixtureMismatch
			if errors.As(err, &e) {
				t.Error(err)
				return
			}
		}

		t.Error(err)
	}
}

// AssertJson compares the actual json data received with expected data in the
// golden files. If the update flag is set, it will also update the golden
// file.
//
// `name` refers to the name of the test and it should typically be unique
// within the package. Also it should be a valid file name (so keeping to
// `a-z0-9\-\_` is a good idea).
func (g *Goldie) AssertJson(t testing.TB, name string, actualJsonData interface{}) {
	t.Helper()
	js, err := json.MarshalIndent(actualJsonData, "", "  ")

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	g.Assert(t, name, normalizeLF(js))
}

// AssertXml compares the actual xml data received with expected data in the
// golden files. If the update flag is set, it will also update the golden
// file.
//
// `name` refers to the name of the test and it should typically be unique
// within the package. Also it should be a valid file name (so keeping to
// `a-z0-9\-\_` is a good idea).
func (g *Goldie) AssertXml(t testing.TB, name string, actualXmlData interface{}) {
	t.Helper()
	x, err := xml.MarshalIndent(actualXmlData, "", "  ")

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	g.Assert(t, name, normalizeLF(x))
}

// normalizeLF normalizes line feed character set across os (es)
// \r\n (windows) & \r (mac) into \n (unix)
func normalizeLF(d []byte) []byte {
	// if empty / nil return as is
	if len(d) == 0 {
		return d
	}
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}

// Assert compares the actual data received with the expected data in the
// golden files after executing it as a template with data parameter. If the
// update flag is set, it will also update the golden file.  `name` refers to
// the name of the test and it should typically be unique within the package.
// Also it should be a valid file name (so keeping to `a-z0-9\-\_` is a good
// idea).
func (g *Goldie) AssertWithTemplate(t testing.TB, name string, data interface{}, actualData []byte) {
	t.Helper()
	if *update {
		var err error
		if *withTemplate {
			err = g.UpdateWithTemplate(t, name, data, actualData)
		} else {
			err = g.Update(t, name, actualData)
		}
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	err := g.compareTemplate(t, name, data, actualData)
	if err != nil {
		{
			var e *errFixtureNotFound
			if errors.As(err, &e) {
				t.Error(err)
				t.FailNow()
				return
			}
		}

		{
			var e *errFixtureMismatch
			if errors.As(err, &e) {
				t.Error(err)
				return
			}
		}

		t.Error(err)
	}
}

// compare is reading the golden fixture file and compare the stored data with
// the actual data.
func (g *Goldie) compare(t testing.TB, name string, actualData []byte) error {
	expectedData, err := g.goldenFileData(t, name)
	if err != nil {
		return err
	}

	if !g.equal(actualData, expectedData) {
		msg := "Result did not match the golden fixture. Diff is below:\n\n"
		actual := string(actualData)
		expected := string(expectedData)

		if g.diffFn != nil {
			msg += g.diffFn(actual, expected)
		} else {
			msg += Diff(g.diffEngine, actual, expected)
		}

		return newErrFixtureMismatch(msg)
	}

	return nil
}

// compareTemplate is reading the golden fixture file and compare the stored
// data with the actual data.
func (g *Goldie) compareTemplate(t testing.TB, name string, data interface{}, actualData []byte) error {
	expectedDataTmpl, err := ioutil.ReadFile(g.GoldenFileName(t, name))

	if err != nil {
		if os.IsNotExist(err) {
			return newErrFixtureNotFound()
		}

		return fmt.Errorf("expected %s to be nil", err.Error())
	}

	missingKey := "error"
	if g.ignoreTemplateErrors {
		missingKey = "default"
	}

	tmpl, err := template.New("test").Option("missingkey=" + missingKey).Parse(string(expectedDataTmpl))
	if err != nil {
		return fmt.Errorf("expected %s to be nil", err.Error())
	}

	var expectedData bytes.Buffer
	err = tmpl.Execute(&expectedData, data)
	if err != nil {
		return newErrMissingKey(fmt.Sprintf("Template error: %s", err.Error()))
	}

	if !g.equal(actualData, expectedData.Bytes()) {
		msg := "Result did not match the golden fixture. Diff is below:\n\n"
		actual := string(actualData)
		expected := expectedData.String()

		if g.diffFn != nil {
			msg += g.diffFn(actual, expected)
		} else {
			msg += Diff(g.diffEngine, actual, expected)
		}

		return newErrFixtureMismatch(msg)
	}

	return nil
}

func (g *Goldie) equal(actual, expected []byte) bool {
	if g.equalFn != nil {
		return g.equalFn(actual, expected)
	}
	return bytes.Equal(actual, expected)
}
