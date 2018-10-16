package main

import (
	"errors"
	"fmt"
	"net"
	"time"

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

	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.host, s.port), time.Second*5)
	if err != nil {
		return err
	}
	s.bs = beanstalk.NewConn(c)
	s.connected = true

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

func (s *server) CurrentTubeName() (string, error) {

	if !s.connected {
		return "", errors.New("Can't determine current tube, not connected to a beanstalk server")
	}

	currentTube := s.CurrentTube()
	return currentTube.Name, nil
}

func (s *server) Delete(toDelete uint64) error {
	return s.bs.Delete(toDelete)
}

func (s *server) DeleteAll(state, name string) (int, error) {
	tube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}

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
}

func (s *server) Disconnect() error {

	if !s.connected {
		return errors.New("Can't disconnect, not connected to a beanstalk server")
	}

	s.connected = false
	return s.bs.Close()
}

func (s *server) GetTubeStats() (map[string]map[string]string, error) {

	if !s.connected {
		return nil, errors.New("Can't get tube stats, not connected to a beanstalk server")
	}

	tubes, err := s.ListTubes()
	if err != nil {
		return nil, err
	}

	var tubeStats = map[string]map[string]string{}
	for _, tube := range tubes {
		stats, _ := s.StatsTube(tube)
		tubeStats[tube] = stats
	}
	return tubeStats, nil
}

func (s *server) isConnected() bool {
	return s.connected
}

func (s *server) Kick(name string, toKick int) (int, error) {

	if !s.connected {
		return 0, errors.New("Can't kick, not connected to a beanstalk server")
	}

	tube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}
	return tube.Kick(toKick)
}

func (s *server) ListTubes() ([]string, error) {

	if !s.connected {
		return nil, errors.New("Can't list tubes, not connected to a beanstalk server")
	}

	tubes, err := s.bs.ListTubes()
	if err != nil {
		return nil, err
	}
	return tubes, nil
}

func (s *server) Peek(state, name string) (uint64, []byte, error) {
	if !s.connected {
		return 0, nil, errors.New("Can't peek, not connected to a beanstalk server")
	}

	tube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}

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

// TODO allow jobs options to be passed
func (s *server) Put(body []byte, name string) (uint64, error) {
	tube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}
	return tube.Put(body, 1, 0, 180)
}

func (s *server) Stats() (map[string]string, error) {

	if !s.connected {
		return nil, errors.New("Can't provide stats, not connected to a beanstalk server")
	}

	return s.bs.Stats()
}

func (s *server) StatsJob(id uint64) (map[string]string, error) {

	if !s.connected {
		return nil, errors.New("Can't stats job, not connected to a beanstalk server")
	}

	return s.bs.StatsJob(id)
}

func (s *server) StatsTube(name string) (map[string]string, error) {

	if !s.connected {
		return nil, errors.New("Can't stats tubes, not connected to a beanstalk server")
	}

	tube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}
	return tube.Stats()
}

func (s *server) UseTube(name string) {
	newTube := beanstalk.Tube{
		Conn: s.bs,
		Name: name,
	}
	s.bs.Tube = newTube
}
