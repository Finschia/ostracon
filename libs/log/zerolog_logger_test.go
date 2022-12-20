package log_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/line/ostracon/libs/log"
)

func TestZeroLogLoggerLogsItsErrors(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:info", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("foo", "baz baz", "bar")

	msg := strings.TrimSpace(buf.String())
	if !strings.Contains(msg, "foo") {
		t.Errorf("expected logger msg to contain ErrInvalidKey, got %s", msg)
	}

	str, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(str), "foo") {
		t.Errorf("expected file logger msg to contain ErrInvalidKey, got %s", msg)
	}
}

func TestZeroLogInfo(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:info", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("Client initialized with old header (trusted is more recent)",
		"old", 42,
		"trustedHeight", "forty two",
		"trustedHash", fmt.Sprintf("%x", []byte("test me")))

	msg := strings.TrimSpace(buf.String())
	msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
	const expectedMsg = `Client initialized with old header (trusted is more recent) old=42 trustedHash=74657374206d65 trustedHeight="forty two"`
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}

	str, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	msg = strings.TrimSpace(string(str))
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}
}

func TestZeroLogDebug(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:debug", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("Client initialized with old header (trusted is more recent)",
		"old", 42,
		"trustedHeight", "forty two",
		"trustedHash", fmt.Sprintf("%x", []byte("test me")))

	msg := strings.TrimSpace(buf.String())
	msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
	const expectedMsg = `Client initialized with old header (trusted is more recent) old=42 trustedHash=74657374206d65 trustedHeight="forty two"`
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}

	str, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	msg = strings.TrimSpace(string(str))
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}
}

func TestZeroLogError(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:error", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}
	logger.Error("Client initialized with old header (trusted is more recent)",
		"old", 42,
		"trustedHeight", "forty two",
		"trustedHash", fmt.Sprintf("%x", []byte("test me")))

	msg := strings.TrimSpace(buf.String())
	msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
	const expectedMsg = `Client initialized with old header (trusted is more recent) old=42 trustedHash=74657374206d65 trustedHeight="forty two"`
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}

	str, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	msg = strings.TrimSpace(string(str))
	if !strings.Contains(msg, expectedMsg) {
		t.Fatalf("received %s, expected %s", msg, expectedMsg)
	}
}

func TestZeroLogLevelForModules(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "module1:debug,module2:info,*:error", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}

	// logger level = debug
	// debug O, info O, error O
	{
		loggerForModule1 := logger.With("module", "module1")

		// debug O
		loggerForModule1.Debug("a1", "b1", "c1")
		msg := strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg := "a1 b1=c1"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// info O
		loggerForModule1.Info("a2", "b2", "c2")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a2 b2=c2"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// error O
		loggerForModule1.Error("a3", "b3", "c3")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a3 b3=c3"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
	}

	// logger level = info
	// debug X, info O, error O
	{
		loggerForModule2 := logger.With("module", "module2")

		// debug X
		loggerForModule2.Debug("a4", "b4", "c4")
		msg := strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg := "a4 b4=c4"
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// info O
		loggerForModule2.Info("a5", "b5", "c5")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a5 b5=c5"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// error O
		loggerForModule2.Error("a6", "b6", "c6")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a6 b6=c6"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
	}

	// logger level = error
	// debug X, info X, error O
	{
		loggerForModule3 := logger.With("module", "module3")

		// debug X
		loggerForModule3.Debug("a7", "b7", "c7")
		msg := strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg := "a7 b7=c7"
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// info X
		loggerForModule3.Info("a8", "b8", "c8")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a8 b8=c8"
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}

		// error O
		loggerForModule3.Error("a9", "b9", "c9")
		msg = strings.TrimSpace(buf.String())
		msg = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(msg, "")
		expectedMsg = "a9 b9=c9"
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
		str, err = ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		msg = strings.TrimSpace(string(str))
		if !strings.Contains(msg, expectedMsg) {
			t.Fatalf("received %s, expected %s", msg, expectedMsg)
		}
	}
}

func TestZeroLogRotateFiles(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		t.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:info", filepath, 1, 1, 2)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		t.Fatal(err)
	}

	// it writes more than 1M logs so that two log files are created
	msg := "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < 6000; i++ {
		logger.Info(msg)
	}

	// check if only two log files is created
	// ex) app.log, app-2022-12-16T10-21-41.295.log
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("file count created %d, expected %d", len(files), 2)
	}

	// change the filename of the backed up log file to one created 2 days ago
	// ex) app-2022-12-16T10-21-41.295.log -> app-2022-12-14T10-21-41.295.log
	now := time.Now().Add(-time.Hour * 48)
	filename2 := fmt.Sprintf("app-%d-%02d-%02dT%02d-%02d-%02d.000.log", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	for _, f := range files {
		if f.Name() == "app.log" {
			continue
		}

		err = os.Rename(dir+"/"+f.Name(), dir+"/"+filename2)
		if err != nil {
			t.Fatalf("file rename failed old filename: %s, new filename: %s", dir+"/"+f.Name(), dir+"/"+filename2)
		}
	}

	// write new logs for rotating log file(remove old log file)
	// ex) app.log, app-2022-12-16T10-21-43.374.log
	for i := 0; i < 6000; i++ {
		logger.Info(msg)
	}

	// check if old log file is removed
	files, err = ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("created %d, expected %d", len(files), 2)
	}
	for _, f := range files {
		if f.Name() == filename2 {
			t.Fatalf("The log file has not been removed yet")
		}
	}
}

func BenchmarkZeroLogLoggerSimple(b *testing.B) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		b.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:info", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		b.Fatal(err)
	}

	benchmarkRunner(b, logger, baseInfoMessage)
}

func BenchmarkZeroLogLoggerContextual(b *testing.B) {
	dir, err := ioutil.TempDir("/tmp", "zerolog-test")
	if err != nil {
		b.Fatal(err)
	}
	filepath := dir + "/app.log"

	var buf bytes.Buffer
	config := log.NewZeroLogConfig(true, "*:info", filepath, 0, 100, 0)
	logger, err := log.NewZeroLogLogger(config, &buf)
	if err != nil {
		b.Fatal(err)
	}

	benchmarkRunner(b, logger, withInfoMessage)
}
