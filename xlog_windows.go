package xlog

import (
	"fmt"
	"strings"

	"github.com/golang/sys/windows"
	"github.com/golang/sys/windows/svc/eventlog"
)

type writer struct {
	pri severity
	src string
	el  *eventlog.Log
}

// Write sends a log message to the Event Log.
func (w *writer) Write(b []byte) (int, error) {
	switch w.pri {
	case sInfo:
		return len(b), w.el.Info(1, string(b))
	case sWarning:
		return len(b), w.el.Warning(3, string(b))
	case sError:
		return len(b), w.el.Error(2, string(b))
	}
	return 0, fmt.Errorf("unrecognized severity: %v", w.pri)
}

func (w *writer) Close() error {
	return w.el.Close()
}

func newW(pri severity, src string) (*writer, error) {
	// Continue if we receive "registry key already exists" or if we get
	// ERROR_ACCESS_DENIED so that we can log without administrative permissions
	// for pre-existing eventlog sources.
	if err := eventlog.InstallAsEventCreate(src, eventlog.Info|eventlog.Warning|eventlog.Error); err != nil {
		if !strings.Contains(err.Error(), "registry key already exists") && err != windows.ERROR_ACCESS_DENIED {
			return nil, err
		}
	}
	el, err := eventlog.Open(src)
	if err != nil {
		return nil, err
	}
	return &writer{
		pri: pri,
		src: src,
		el:  el,
	}, nil
}

func setup(src string) (*writer, *writer, *writer, error) {
	infoL, err := newW(sInfo, src)
	if err != nil {
		return nil, nil, nil, err
	}
	warningL, err := newW(sWarning, src)
	if err != nil {
		return nil, nil, nil, err
	}
	errL, err := newW(sError, src)
	if err != nil {
		return nil, nil, nil, err
	}
	return infoL, warningL, errL, nil
}
