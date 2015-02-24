package logmux

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/technoweenie/grohl"

	"github.com/flynn/flynn/discoverd/client"
	"github.com/flynn/flynn/pkg/syslog/rfc5424"
	"github.com/flynn/flynn/pkg/syslog/rfc6587"
)

// LogMux collects log lines from multiple readers and forwards them to a log
// aggregator service registered in discoverd. Log lines are buffered in memory
// and are dropped in LIFO order.
type LogMux struct {
	logc chan *rfc5424.Message

	producerwg *sync.WaitGroup

	shutdowno sync.Once
	shutdownc chan struct{}

	doneo sync.Once
	donec chan struct{}
}

// New returns an instance of LogMux ready to follow io.Reader producers. The
// log messages are buffered internally until Connect is called.
func New(bufferSize int) *LogMux {
	return &LogMux{
		logc:       make(chan *rfc5424.Message, bufferSize),
		producerwg: &sync.WaitGroup{},
		shutdownc:  make(chan struct{}),
		donec:      make(chan struct{}),
	}
}

// Connect opens a connection to the named log aggregation service in discoverd
// and creates a goroutine that drains the log message buffer to the connection.
func (m *LogMux) Connect(discd *discoverd.Client, name string) error {
	conn, err := connect(discd, name, m.donec)
	if err != nil {
		return err
	}

	go m.drainTo(conn)
	return nil
}

func (m *LogMux) drainTo(w io.Writer) {
	defer close(m.donec)

	g := grohl.NewContext(grohl.Data{"at": "logmux_drain"})

	for {
		msg, ok := <-m.logc
		if !ok {
			return // shutdown
		}

		_, err := w.Write(rfc6587.Bytes(msg))
		if err != nil {
			g.Log(grohl.Data{"status": "error", "err": err.Error()})

			// write logs to local logger when the writer fails
			g.Log(grohl.Data{"msg": msg.String()})
			for msg := range m.logc {
				g.Log(grohl.Data{"msg": msg.String()})
			}

			return // shutdown
		}
	}
}

// Close blocks until all producers have finished, then terminates the drainer,
// and blocks until the backlog in logc has been processed.
func (m *LogMux) Close() {
	m.producerwg.Wait()

	m.doneo.Do(func() { close(m.logc) })
	<-m.donec
}

type Config struct {
	AppName, IP, JobType, JobID string
}

// Follow forwards log lines from the reader into the syslog client. Follow
// runs until the reader is closed or an error occurs. If an error occurs, the
// reader may still be open.
func (m *LogMux) Follow(r io.Reader, fd int, config Config) {
	m.producerwg.Add(1)

	if config.AppName == "" {
		config.AppName = config.JobID
	}

	hdr := &rfc5424.Header{
		Hostname: []byte(config.IP),
		AppName:  []byte(config.AppName),
		ProcID:   []byte(config.JobType + "." + config.JobID),
		MsgID:    []byte(fmt.Sprintf("ID%d", fd)),
	}

	go m.follow(r, hdr)
}

func (m *LogMux) follow(r io.Reader, hdr *rfc5424.Header) {
	defer m.producerwg.Done()

	g := grohl.NewContext(grohl.Data{"at": "logmux_follow"})
	bufr := bufio.NewReader(r)

	for {
		line, _, err := bufr.ReadLine()
		if err == io.EOF {
			return
		}
		if err != nil {
			g.Log(grohl.Data{"status": "error", "err": err.Error()})
			return
		}

		msg := rfc5424.NewMessage(hdr, line)

		select {
		case m.logc <- msg:
		default:
			// throw away msg if logc buffer is full
		}
	}
}
