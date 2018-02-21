package main

import (
	"errors"
	"fmt"

	"github.com/kr/beanstalk"
)

type server struct {
	bs        *beanstalk.Conn
	connected bool
	host      string
	port      int
}

func (s *server) connect() (err error) {
	if s.host == "" {
		s.host = "127.0.0.1"
	}

	if s.port == 0 {
		s.port = 11300
	}

	s.bs, err = beanstalk.Dial("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err == nil {
		s.connected = true
	}
	return
}

func (s *server) Bury(toBury uint64) error {
	return s.bs.Bury(toBury, 1)
}

func (s *server) Connect(host string, port int) error {
	s.host = host
	s.port = port
	return s.connect()
}

func (s *server) ConnectionStr() (string, error) {
	if s.connected {
		return fmt.Sprintf("%v:%v", s.host, s.port), nil
	}

	return "", errors.New("not connected to a beanstalk server")
}

func (s *server) CurrentTube() *beanstalk.Tube {
	return &s.bs.Tube
}

func (s *server) CurrentTubeName() string {
	currentTube := s.CurrentTube()
	return currentTube.Name
}

func (s *server) Delete(toDelete uint64) error {
	return s.bs.Delete(toDelete)
}

func (s *server) DeleteAll(state, name string) (int, error) {
	tube := beanstalk.Tube{s.bs, name}

	var (
		err error
		id  uint64
		n   int
	)

	for {
		switch state {
		case "ready":
			id, _, err = tube.PeekReady()
		case "buried":
			id, _, err = tube.PeekBuried()
		case "delayed":
			id, _, err = tube.PeekReady()
		}
		if err != nil {
			return n, err
		}

		if err := s.Delete(id); err != nil {
			return n, err
		}

		n++
	}

	return n, err
}

func (s *server) Disconnect() error {
	s.connected = false
	return s.bs.Close()
}

func (s *server) GetTubeStats() map[string]map[string]string {
	var tubes = map[string]map[string]string{}
	for _, tube := range s.ListTubes() {
		stats, _ := s.StatsTube(tube)
		tubes[tube] = stats
	}
	return tubes
}

func (s *server) isConnected() bool {
	return s.connected
}

func (s *server) Kick(name string, toKick int) (int, error) {
	tube := beanstalk.Tube{s.bs, name}
	return tube.Kick(toKick)
}

func (s *server) ListTubes() []string {
	tubes, _ := s.bs.ListTubes()
	return tubes
}

func (s *server) Peek(state, name string) (uint64, []byte, error) {
	tube := beanstalk.Tube{s.bs, name}

	var (
		id   uint64
		body []byte
		err  error
	)
	switch state {
	case "buried":
		id, body, err = tube.PeekBuried()
	case "delayed":
		id, body, err = tube.PeekDelayed()
	case "ready":
		id, body, err = tube.PeekReady()
	}

	return id, body, err
}

func (s *server) Stats() (map[string]string, error) {
	return s.bs.Stats()
}

func (s *server) StatsTube(name string) (map[string]string, error) {
	tube := beanstalk.Tube{s.bs, name}
	return tube.Stats()
}

func (s *server) UseTube(name string) {
	newTube := beanstalk.Tube{s.bs, name}
	s.bs.Tube = newTube
}
