package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type Room struct {
	Name        string
	Description string
	Exits       map[string]string // direction -> room name
}

type Player struct {
	Name string
	Room string
	Conn net.Conn
}

var (
	rooms   map[string]*Room
	players map[net.Conn]*Player
	mu      sync.RWMutex
)

func init() {
	rooms = map[string]*Room{
		"entrance": {
			Name:        "Entrance Hall",
			Description: "A dimly lit stone hall. Torches flicker on the walls.",
			Exits:       map[string]string{"north": "corridor", "east": "armory"},
		},
		"corridor": {
			Name:        "Dark Corridor",
			Description: "A narrow passage stretching into darkness. You hear dripping water.",
			Exits:       map[string]string{"south": "entrance", "north": "throne"},
		},
		"armory": {
			Name:        "Old Armory",
			Description: "Rusted weapons line the walls. A broken shield lies on the floor.",
			Exits:       map[string]string{"west": "entrance"},
		},
		"throne": {
			Name:        "Throne Room",
			Description: "A grand hall with a crumbling throne. Dust motes float in shafts of light.",
			Exits:       map[string]string{"south": "corridor"},
		},
	}
	players = make(map[net.Conn]*Player)
}

func main() {
	ln, err := net.Listen("tcp", ":9983")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("mud listening on :9983")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Fprint(conn, "Welcome to the MUD!\r\nWhat is your name? ")

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		name = "stranger"
	}

	player := &Player{Name: name, Room: "entrance", Conn: conn}
	mu.Lock()
	players[conn] = player
	mu.Unlock()

	broadcast(fmt.Sprintf("%s has entered the world.", name), conn)
	sendRoom(player)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		handleCommand(player, line)
	}

	mu.Lock()
	delete(players, conn)
	mu.Unlock()
	broadcast(fmt.Sprintf("%s has left.", name), nil)
}

func handleCommand(p *Player, input string) {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "look", "l":
		sendRoom(p)
	case "north", "south", "east", "west", "n", "s", "e", "w":
		dir := cmd
		switch dir {
		case "n":
			dir = "north"
		case "s":
			dir = "south"
		case "e":
			dir = "east"
		case "w":
			dir = "west"
		}
		move(p, dir)
	case "say":
		if len(parts) < 2 {
			fmt.Fprint(p.Conn, "Say what?\r\n")
			return
		}
		msg := parts[1]
		broadcastRoom(p.Room, fmt.Sprintf("%s says: %s", p.Name, msg), nil)
	case "who":
		mu.RLock()
		for _, other := range players {
			fmt.Fprintf(p.Conn, "  %s (%s)\r\n", other.Name, rooms[other.Room].Name)
		}
		mu.RUnlock()
	case "quit":
		fmt.Fprint(p.Conn, "Goodbye!\r\n")
		p.Conn.Close()
	case "help":
		fmt.Fprint(p.Conn, "Commands: look, north/south/east/west (n/s/e/w), say <msg>, who, quit, help\r\n")
	default:
		fmt.Fprintf(p.Conn, "Unknown command: %s (type 'help')\r\n", cmd)
	}
}

func move(p *Player, dir string) {
	mu.RLock()
	room := rooms[p.Room]
	mu.RUnlock()

	dest, ok := room.Exits[dir]
	if !ok {
		fmt.Fprintf(p.Conn, "You can't go %s.\r\n", dir)
		return
	}

	broadcastRoom(p.Room, fmt.Sprintf("%s goes %s.", p.Name, dir), p.Conn)
	p.Room = dest
	broadcastRoom(p.Room, fmt.Sprintf("%s arrives.", p.Name), p.Conn)
	sendRoom(p)
}

func sendRoom(p *Player) {
	mu.RLock()
	room := rooms[p.Room]
	mu.RUnlock()

	fmt.Fprintf(p.Conn, "\r\n== %s ==\r\n%s\r\n", room.Name, room.Description)
	exits := make([]string, 0, len(room.Exits))
	for dir := range room.Exits {
		exits = append(exits, dir)
	}
	fmt.Fprintf(p.Conn, "Exits: %s\r\n", strings.Join(exits, ", "))

	mu.RLock()
	for _, other := range players {
		if other.Conn != p.Conn && other.Room == p.Room {
			fmt.Fprintf(p.Conn, "%s is here.\r\n", other.Name)
		}
	}
	mu.RUnlock()
	fmt.Fprint(p.Conn, "> ")
}

func broadcast(msg string, exclude net.Conn) {
	mu.RLock()
	defer mu.RUnlock()
	for conn := range players {
		if conn != exclude {
			fmt.Fprintf(conn, "%s\r\n", msg)
		}
	}
}

func broadcastRoom(roomName, msg string, exclude net.Conn) {
	mu.RLock()
	defer mu.RUnlock()
	for conn, p := range players {
		if conn != exclude && p.Room == roomName {
			fmt.Fprintf(conn, "%s\r\n> ", msg)
		}
	}
}
