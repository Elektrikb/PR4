package models

import "encoding/xml"

type GameResult struct {
	XMLName    xml.Name  `xml:"game"`
	StartTime  string    `xml:"start_time"`
	EndTime    string    `xml:"end_time"`
	SecretCode string    `xml:"secret_code"`
	Players    []*Player `xml:"players>player"`
	Winner     *int      `xml:"winner,omitempty"`
}
