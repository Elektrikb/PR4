package handlers

import (
	"encoding/json"
	"fmt"
	"game/game"
	"game/models"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type Handler struct {
	mu           sync.Mutex
	games        map[int]*game.Game
	gameCounter  int
	startingGame bool
	re           *regexp.Regexp
}

func NewHandler() *Handler {
	handler := &Handler{
		mu:           sync.Mutex{},
		gameCounter:  0,
		games:        make(map[int]*game.Game),
		startingGame: false,
	}

	re, _ := regexp.Compile("^[A-Z0-9]{4}$")
	handler.re = re

	return handler
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.startingGame {
		h.gameCounter++
		h.startingGame = true
		newGame := game.NewGame()
		h.games[h.gameCounter] = newGame

		log.Printf("Секретный код игры %d: %s", h.gameCounter, h.games[h.gameCounter].SecretCode)
	}

	game := h.games[h.gameCounter]
	playerId, err := game.AddPlayer()

	if err != nil {
		log.Println("Ошибка:", err.Error())
		return
	}

	response := models.JoinResponse{
		Message:   "Вы успешно присоединились к игре",
		Attempts:  game.MaxAttempts,
		TimeLimit: game.TimeLimit,
		PlayerID:  playerId,
		GameID:    h.gameCounter,
	}

	if playerId == 4 {
		h.startingGame = false
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Guess(w http.ResponseWriter, r *http.Request) {
	guess := r.URL.Query().Get("guess")
	playerId := r.URL.Query().Get("player_id")
	gameId := r.URL.Query().Get("game_id")

	log.Printf("Игра %s. Игрок %s. Отправил догадку %s.\n", gameId, playerId, guess)

	playerIdInt, err := strconv.Atoi(playerId)
	if err != nil {
		response := models.GuessResponse{
			Message: "Некорректный идентификатор игры",
		}

		log.Printf("Игра %s. Игрок %s. Некорректный идентификатор игры %s.\n", gameId, playerId, gameId)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

		return
	}

	gameIdInt, err := strconv.Atoi(gameId)
	if err != nil {
		response := models.GuessResponse{
			Message: "Некорректный идентификатор игрока",
		}

		log.Printf("Игра %s. Игрок %s. Некорректный идентификатор игрока %s.\n", gameId, playerId, playerId)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

		return
	}

	game, ok := h.games[gameIdInt]

	if !ok {
		response := models.GuessResponse{
			Message: "Игры не существует",
		}

		log.Printf("Игра %s. Игрок %s. Игры %s не существует.\n", gameId, playerId, gameId)

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)

		return
	}

	if game.IsEnd {
		response := models.GuessResponse{
			IsEnd:   &game.IsEnd,
			Message: "Игра окончена",
		}

		log.Printf("Игра %s. Игрок %s. Игра %s окончена.\n", gameId, playerId, gameId)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

		return
	}

	if !h.validateGuess(guess) {
		response := models.GuessResponse{
			Message: "Некорректная догадка",
		}

		log.Printf("Игра %s. Игрок %s. Некорректная догадка %s.\n", gameId, playerId, guess)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

		return
	}

	player := game.Players[playerIdInt]

	black, white := game.CheckGuess(guess)

	game.Mu.Lock()
	defer game.Mu.Unlock()

	player.Attempt++
	remainingTime := time.Until(game.StartTime.Add(game.TimeLimit))

	switch {
	case remainingTime <= 0:
		game.IsEnd = true
		game.EndTime = time.Now()
		gameWinner := 0
		game.Winner = &gameWinner
		game.SaveResult()

		response := models.GuessResponse{
			IsEnd:   &game.IsEnd,
			Message: fmt.Sprintf("Время вышло. Секретный код был %s", game.SecretCode),
		}

		log.Printf("Игра %s. Игрок %s. Время вышло.\n", gameId, playerId)

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

	case black == game.CodeLength:
		game.IsEnd = true
		game.EndTime = time.Now()
		game.Winner = &playerIdInt
		game.SaveResult()

		response := models.GuessResponse{
			IsEnd:   &game.IsEnd,
			Message: "Поздравляем! Вы угадали код!",
		}

		log.Printf("Игра %s. Игрок %s. Код угадан.\n", gameId, playerId)

		json.NewEncoder(w).Encode(response)

	case player.Attempt >= game.MaxAttempts:
		playerIsEnd := true

		response := models.GuessResponse{
			IsEnd:   &playerIsEnd,
			Message: fmt.Sprintf("Превышено максимально допустимое количество попыток. Секретный код был %s", game.SecretCode),
		}

		log.Printf("Игра %s. Игрок %s. Превышено максимально допустимое количество попыток.\n", gameId, playerId)

		attemptsOver := true
		for _, player := range game.Players {
			if player.Attempt < game.MaxAttempts {
				attemptsOver = false
				break
			}
		}

		if attemptsOver {
			gameWinner := 0
			game.Winner = &gameWinner
			game.EndTime = time.Now()
			game.SaveResult()
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)

	default:
		response := models.GuessResponse{
			IsEnd:       &game.IsEnd,
			Message:     "Ответ принят",
			Attempt:     &player.Attempt,
			MaxAttempts: &game.MaxAttempts,
			Black:       &black,
			White:       &white,
			TimeLeft:    &remainingTime,
		}

		log.Printf("Игра %s. Игрок %s. Черные %d, белые %d.\n", gameId, playerId, black, white)

		json.NewEncoder(w).Encode(response)

	}

}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	gameId := r.URL.Query().Get("game_id")
	gameIdInt, _ := strconv.Atoi(gameId)

	game := h.games[gameIdInt]

	h.mu.Lock()
	defer h.mu.Unlock()
	game.Mu.Lock()
	defer game.Mu.Unlock()

	if game.IsStart {
		response := models.StatusResponse{
			IsBegin: true,
			Message: "Игра началась",
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	switch len(game.Players) {
	case 1:
		response := models.StatusResponse{
			IsBegin: false,
			Message: "Ожидание игроков (1/4).",
		}

		json.NewEncoder(w).Encode(response)
	case 2, 3:
		remainingTime := time.Until(game.WaitTime.Add(game.TimeBeforeStart))
		if remainingTime > 0 {
			response := models.StatusResponse{
				IsBegin: false,
				Message: fmt.Sprintf("Ожидание игроков (%d/4). Осталось %v", len(game.Players), remainingTime),
			}

			json.NewEncoder(w).Encode(response)
		} else {
			response := models.StatusResponse{
				IsBegin: true,
				Message: "Игра началась",
			}

			h.startingGame = false
			game.StartTime = time.Now()
			game.IsStart = true

			json.NewEncoder(w).Encode(response)
		}
	case 4:
		remainingTime := time.Until(game.WaitTime.Add(game.TimeBeforeStart))
		if remainingTime > 0 {
			response := models.StatusResponse{
				IsBegin: false,
				Message: fmt.Sprintf("Ожидание игроков (4/4). Осталось %v", remainingTime),
			}

			json.NewEncoder(w).Encode(response)
		} else {
			response := models.StatusResponse{
				IsBegin: true,
				Message: "Игра началась",
			}

			game.StartTime = time.Now()
			game.IsStart = true

			json.NewEncoder(w).Encode(response)
		}

	}

}

func (h *Handler) validateGuess(guess string) bool {
	return h.re.MatchString(guess)
}
