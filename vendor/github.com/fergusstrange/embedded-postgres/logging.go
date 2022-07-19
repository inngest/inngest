package embeddedpostgres

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type syncedLogger struct {
	offset int64
	logger io.Writer
	file   *os.File
}

func newSyncedLogger(dir string, logger io.Writer) (*syncedLogger, error) {
	file, err := ioutil.TempFile(dir, "embedded_postgres_log")
	if err != nil {
		return nil, err
	}

	s := syncedLogger{
		logger: logger,
		file:   file,
	}

	return &s, nil
}

func (s *syncedLogger) flush() error {
	if s.logger != nil {
		file, err := os.Open(s.file.Name())
		if err != nil {
			return fmt.Errorf("unable to process postgres logs: %s", err)
		}

		defer func() {
			if err := file.Close(); err != nil {
				panic(err)
			}
		}()

		if _, err = file.Seek(s.offset, io.SeekStart); err != nil {
			return fmt.Errorf("unable to process postgres logs: %s", err)
		}

		readBytes, err := io.Copy(s.logger, file)
		if err != nil {
			return fmt.Errorf("unable to process postgres logs: %s", err)
		}

		s.offset += readBytes
	}

	return nil
}
