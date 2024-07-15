package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"goredis/client"

	"github.com/labstack/gommon/log"
)

const defaultListenAddr = ":5001"

type Config struct {
	ListenAddr string
}

type Message struct {
	data []byte
	peer *Peer
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message

	//
	kv *KV
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}

	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan Message),
		kv:        NewKV(),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)

	if err != nil {
		return err
	}

	s.ln = ln

	go s.loop()

	slog.Info("server running", "listenAddr", s.ListenAddr)

	return s.acceptLoop()
}

func (s *Server) handleMessage(msg Message) error {
	cmd, err := parseCommand(string(msg.data))

	if err != nil {
		return err
	}

	switch v := cmd.(type) {
	case GetCommand:
		val, ok := s.kv.Get(v.key)

		if !ok {
			return fmt.Errorf("key not found: %s", v.key)
		}

		_, err := msg.peer.Send(val)

		if err != nil {
			slog.Error("peer send error", "err", err)
		}

	case SetCommand:
		return s.kv.Set(v.key, v.val)
	}

	return nil
}

func (s *Server) loop() {
	for {
		select {
		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				slog.Error("raw message error", "err", err)
			}
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		case <-s.quitCh:
			return
		}
	}

}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()

		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh) // Create new peer
	s.addPeerCh <- peer            // Add peer to peers map

	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", "err", err, "remoteAddr", conn.RemoteAddr().String())
	}
}

func main() {
	server := NewServer(Config{})

	go func() {
		log.Fatal(server.Start())
	}()

	time.Sleep(time.Second)

	client := client.New("localhost:5001")
	for i := 0; i < 10; i++ {

		if err := client.Set(context.Background(), fmt.Sprintf("foo_%d", i), "bar"); err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Second)

		val, err := client.Get(context.Background(), fmt.Sprintf("foo_%d", i))

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("got this back =>", val)
	}
}
