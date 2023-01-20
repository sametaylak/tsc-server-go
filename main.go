package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type Message struct {
	FileName string
	Line     int
	Column   int
	Message  string
}

type Server struct {
	RootFolderPath string
	Host           string
	Port           int
	Connections    []net.Conn
	Messages       []Message
}

func (s *Server) Run() error {
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		s.AddConnection(conn)
		go handleRequest(conn, s)
	}
}

func (s *Server) AddConnection(conn net.Conn) error {
	s.Connections = append(s.Connections, conn)

	return nil
}

func (s *Server) RemoveConnection(conn net.Conn) error {
	for i, c := range s.Connections {
		if c == conn {
			s.Connections = append(s.Connections[:i], s.Connections[i+1:]...)

			return nil
		}
	}

	return nil
}

func (s *Server) AddMessage(m Message) error {
	s.Messages = append(s.Messages, m)

	return nil
}

func (s *Server) RemoveMessages() error {
	s.Messages = []Message{}

	return nil
}

func (s *Server) SendDataToAll(d []byte) error {
	for _, c := range s.Connections {
		c.Write(d)
	}

	return nil
}

func handleRequest(conn net.Conn, s *Server) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		line, _, err := reader.ReadLine()
		if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
			log.Println(err)
			break
		}

		if err != nil {
			log.Fatal(err)
		}
		parsedLine := string(line)
		if parsedLine == "exit" {
			break
		}

		fmt.Println(string(line))
	}

	s.RemoveConnection(conn)
}

func runTSC(s *Server) {
	re := regexp.MustCompile(`(.*)\(([0-9]+),([0-9]+)\):\s(.*)`)
	r, w := io.Pipe()

	tsc := exec.Command("tsc", "-p", ".", "--watch", "--noEmit")
	tsc.Dir = s.RootFolderPath
	tsc.Stdout = w

	if err := tsc.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		fmt.Println("TSC started")
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			input := scanner.Text()

			// Send collected messages to the clients
			// Below message indicates that tsc is finished compiling
			if strings.Contains(input, "Watching for file changes") {
				p, err := json.Marshal(s.Messages)
				if err != nil {
					log.Fatal(err)
				}
				s.SendDataToAll(p)
				s.RemoveMessages()
				continue
			}

			// Collect messages from tsc
			matches := re.FindStringSubmatch(input)
			if len(matches) > 0 {
				parsedLine, err := strconv.Atoi(matches[2])
				if err != nil {
					log.Fatal(err)
				}

				parsedColumn, err := strconv.Atoi(matches[3])
				if err != nil {
					log.Fatal(err)
				}

				m := Message{
					FileName: matches[1],
					Line:     parsedLine,
					Column:   parsedColumn,
					Message:  matches[4],
				}

				s.AddMessage(m)
			}
		}
	}()
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You should provide a root folder path")
	}

	rootFolderPath := os.Args[1]
	if _, err := os.Stat(rootFolderPath); os.IsNotExist(err) {
		log.Fatal("Invalid root folder path")
	}

	s := Server{
		RootFolderPath: rootFolderPath,
		Host:           "localhost",
		Port:           9000,
	}

	runTSC(&s)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
