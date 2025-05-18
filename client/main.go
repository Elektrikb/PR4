package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const serverURL = "http://localhost:8080"

type JoinResponse struct {
	Message   string        `json:"message"`
	Attempts  int           `json:"attempts"`
	TimeLimit time.Duration `json:"time_limit"`
	PlayerID  int           `json:"player_id"`
	GameID    int           `json:"game_id"`
}

type StatusResponse struct {
	IsBegin bool   `json:"is_begin"`
	Message string `json:"message"`
}

type GuessResponse struct {
	Message     string         `json:"message"`
	Attempt     *int           `json:"attempt,omitempty"`
	MaxAttempts *int           `json:"max_attempts,omitempty"`
	Black       *int           `json:"black,omitempty"`
	White       *int           `json:"white,omitempty"`
	IsEnd       *bool          `json:"is_end,omitempty"`
	TimeLeft    *time.Duration `json:"time_left,omitempty"`
}

var (
	ans            string
	guess          string
	joinResponse   JoinResponse
	statusResponse StatusResponse
	guessResponse  GuessResponse
)

func main() {
	fmt.Println("Добро пожаловать в игру \"Код мастер\"")

	for {
		fmt.Println("Начать новую игру? (да/нет)")

		fmt.Scanln(&ans)
		if ans != "да" {
			break
		}

		resp, err := http.Get(serverURL + "/join")
		if err != nil {
			fmt.Println("Ошибка входа в игру:", err)
			continue
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&joinResponse); err != nil {
			fmt.Printf("Ошибка при чтении ответа сервера: %v\n", err)
			continue
		}

		fmt.Printf("Игрок %v. Игра %v.\n", joinResponse.PlayerID, joinResponse.GameID)
		fmt.Printf("%s. У вас %d попыток и %v времени.\n", joinResponse.Message, joinResponse.Attempts, joinResponse.TimeLimit)

		statusUrl := fmt.Sprintf("%s/status?game_id=%d", serverURL, joinResponse.GameID)

		for {
			resp, err := http.Get(statusUrl)
			if err != nil {
				fmt.Println("Ошибка получения статуса игры:", err)
				return
			}

			fmt.Print("\r")
			fmt.Print("                                             ")

			err = json.NewDecoder(resp.Body).Decode(&statusResponse)
			if err != nil {
				fmt.Printf("Ошибка при чтении ответа сервера: %v\n", err)
				return
			}
			defer resp.Body.Close()

			fmt.Print("\r")
			fmt.Print(statusResponse.Message)
			if statusResponse.IsBegin {
				fmt.Println("")
				break
			}
			time.Sleep(1 * time.Second)
		}

		for {
			fmt.Println("Введите догадку (4 символа A-Z, 0-9)")
			fmt.Scanln(&guess)

			url := fmt.Sprintf(
				"%s/guess?guess=%s&player_id=%v&game_id=%v",
				serverURL,
				guess,
				joinResponse.PlayerID,
				joinResponse.GameID,
			)

			resp, err := http.Get(url)
			if err != nil {
				fmt.Println("Ошибка.", err)
				continue
			}
			defer resp.Body.Close()

			json.NewDecoder(resp.Body).Decode(&guessResponse)

			if guessResponse.IsEnd != nil && *guessResponse.IsEnd {
				fmt.Println(guessResponse.Message)
				break
			}

			if resp.StatusCode != 200 {
				fmt.Println(guessResponse.Message)
				continue
			}

			fmt.Println(guessResponse.Message)
			fmt.Printf("Попытка %d/%d\n", *guessResponse.Attempt, *guessResponse.MaxAttempts)
			fmt.Printf("Осталось времени %v\n", guessResponse.TimeLeft)
			fmt.Printf("Черные: %d, Белые: %d\n", *guessResponse.Black, *guessResponse.White)

		}
	}
}
