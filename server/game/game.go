package game

import (
	"encoding/xml"
	"errors"
	"fmt"
	"game/models"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	MaxAttempts     = 10
	CodeLength      = 4
	TimeBeforeStart = 30 * time.Second
	TimeLimit       = 2 * time.Minute
)

type Game struct {
	Mu              sync.Mutex
	Players         map[int]*models.Player
	Winner          *int
	PlayersCount    int
	CreateTime      time.Time
	WaitTime        time.Time
	StartTime       time.Time
	EndTime         time.Time
	CodeLength      int
	SecretCode      string
	MaxAttempts     int
	TimeBeforeStart time.Duration
	TimeLimit       time.Duration
	IsStart         bool
	IsEnd           bool
}

func NewGame() *Game {
	game := &Game{
		Mu:              sync.Mutex{},
		Players:         make(map[int]*models.Player),
		Winner:          nil,
		PlayersCount:    0,
		CreateTime:      time.Now(),
		CodeLength:      CodeLength,
		MaxAttempts:     MaxAttempts,
		TimeBeforeStart: TimeBeforeStart,
		TimeLimit:       TimeLimit,
		IsStart:         false,
		IsEnd:           false,
	}

	game.generateSecretCode()

	return game
}

func (g *Game) generateSecretCode() {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, g.CodeLength)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}

	g.SecretCode = string(code)
}

func (g *Game) CheckGuess(guess string) (int, int) {
	black, white := 0, 0
	secretCodeMap := make(map[byte]int)
	checked := make([]bool, len(g.SecretCode))

	for i := 0; i < len(g.SecretCode); i++ {
		if g.SecretCode[i] == guess[i] {
			black++
			checked[i] = true
		} else {
			secretCodeMap[g.SecretCode[i]]++
		}
	}

	for i := 0; i < len(guess); i++ {
		if !checked[i] && secretCodeMap[guess[i]] > 0 {
			white++
			secretCodeMap[guess[i]]--
		}
	}

	return black, white
}

func (g *Game) SaveResult() {
	var players []*models.Player

	for _, player := range g.Players {
		players = append(players, player)
	}

	result := models.GameResult{
		StartTime:  g.StartTime.Format(time.RFC3339),
		EndTime:    g.EndTime.Format(time.RFC3339),
		SecretCode: g.SecretCode,
		Players:    players,
		Winner:     g.Winner,
	}

	xmlData, err := xml.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Printf("Ошибка форматирования: %v", err)
		return
	}

	filename := fmt.Sprintf("game_result_%s_%v.xml", g.StartTime.Format("20060102_150405"), time.Now().Unix())
	err = os.WriteFile(filename, xmlData, 0644)
	if err != nil {
		log.Printf("Ошибка записи: %v\n", err)
		return
	}

	log.Printf("Результат игры сохранен в %s\n", filename)
}

func (g *Game) AddPlayer() (int, error) {
	g.PlayersCount++

	if g.PlayersCount == 2 {
		g.WaitTime = time.Now()
	}

	if g.PlayersCount > 4 {
		return 0, errors.New("больше четырех игроков в игре")
	}

	player := &models.Player{
		ID:      g.PlayersCount,
		Attempt: 0,
	}

	g.Players[g.PlayersCount] = player

	return g.PlayersCount, nil
}
