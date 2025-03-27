package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://challenge.z2o.cloud/challenges"

type ChallengeResponse struct {
	ID        string                 `json:"id"`
	ActivesAt int64                  `json:"actives_at"`
	CalledAt  int64                  `json:"called_at"`
	TotalDiff int                    `json:"total_diff"`
	Result    map[string]interface{} `json:"result"`
}

func waitUntil(targetMs int64) {
	targetTime := time.UnixMilli(targetMs)
	sleep := time.Until(targetTime)

	if sleep > 10*time.Millisecond {
		time.Sleep(sleep - 2*time.Millisecond)
	}
	for time.Now().Before(targetTime) {
		// busy wait
	}
}

func saveJSON(path string, v interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func main() {
	nickname := flag.String("nickname", "", "your nickname")
	flag.Parse()

	if *nickname == "" {
		fmt.Println("nickname is required")
		os.Exit(1)
	}

	client := &http.Client{}

	// POST
	postURL := fmt.Sprintf("%s?nickname=%s", baseURL, *nickname)
	resp, err := client.Post(postURL, "application/json", nil)
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var challenge ChallengeResponse
	json.Unmarshal(body, &challenge)

	// post.json 保存（最初の1回）
	saveJSON("post.json", challenge)

	fmt.Printf("⏳ 初回待機 actives_at: %d\n", challenge.ActivesAt)
	waitUntil(challenge.ActivesAt)

	var finalResult *ChallengeResponse = nil

	for {
		req, err := http.NewRequest(http.MethodPut, baseURL, nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("X-Challenge-Id", challenge.ID)

		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result ChallengeResponse
		json.Unmarshal(body, &result)

		if result.Result != nil {
			fmt.Println("✅ result が返ってきました！ループ終了")
			finalResult = &result
			break
		}

		// 少し早めに次のactives_atに送る
		offset := -2 * time.Millisecond
		adjusted := result.ActivesAt + offset.Milliseconds()
		waitUntil(adjusted)
	}

	// 最後の結果だけ put.json に保存
	if finalResult != nil {
		saveJSON("put.json", finalResult)
		fmt.Println("💾 最後のresultのみ put.json に保存しました")
	}

	fmt.Printf("終了")
}
