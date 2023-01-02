package main

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "sync"
    "time"
)

type Crawler struct {
    mu         sync.Mutex
    cfg        *Config
    state      *State
    channelIdx uint32
}

type CurrentLiveResponse struct {
	Movie struct {
		ID               string `json:"id"`
		UserID           string `json:"user_id"`
		Title            string `json:"title"`
		Subtitle         string `json:"subtitle"`
		LastOwnerComment string `json:"last_owner_comment"`
		Category         string `json:"category"`
		Link             string `json:"link"`
		IsLive           bool   `json:"is_live"`
		IsRecorded       bool   `json:"is_recorded"`
		CommentCount     int    `json:"comment_count"`
		LargeThumbnail   string `json:"large_thumbnail"`
		SmallThumbnail   string `json:"small_thumbnail"`
		Country          string `json:"country"`
		Duration         int    `json:"duration"`
		Created          int    `json:"created"`
		IsCollabo        bool   `json:"is_collabo"`
		IsProtected      bool   `json:"is_protected"`
		MaxViewCount     int    `json:"max_view_count"`
		CurrentViewCount int    `json:"current_view_count"`
		TotalViewCount   int    `json:"total_view_count"`
		HlsURL           string `json:"hls_url"`
	} `json:"movie"`
	Broadcaster struct {
		ID              string `json:"id"`
		ScreenID        string `json:"screen_id"`
		Name            string `json:"name"`
		Image           string `json:"image"`
		Profile         string `json:"profile"`
		Level           int    `json:"level"`
		LastMovieID     string `json:"last_movie_id"`
		IsLive          bool   `json:"is_live"`
		SupporterCount  int    `json:"supporter_count"`
		SupportingCount int    `json:"supporting_count"`
		Created         int    `json:"created"`
	} `json:"broadcaster"`
	Tags []string `json:"tags"`
}

func (c *Crawler) nextChannel() *Channel {
    c.mu.Lock()
    defer c.mu.Unlock()
    channel := &c.cfg.Channels[c.channelIdx]
    c.channelIdx = (c.channelIdx + 1) % uint32(len(c.cfg.Channels))
    return channel
}

func (c *Crawler) fetchNext(app *Credentials) (*Channel, *CurrentLiveResponse, error) {
    channel := c.nextChannel()

    log.Printf("Checking channel %s", channel.Name)

    auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", app.ClientID, app.ClientSecret)))
    auth = "Basic " + auth

    url := fmt.Sprintf("https://apiv2.twitcasting.tv/users/%s/current_live", channel.Name)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return channel, nil, fmt.Errorf("Unable to create request: %w", err)
    }
    req.Header.Set("Authorization", auth)
    req.Header.Set("X-Api-Version", "2.0")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return channel, nil, fmt.Errorf("Unable to fetch current live status: %w", err)
    }
    defer resp.Body.Close()

    //not live
    if resp.StatusCode == 404 {
        return channel, nil, nil
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return channel, nil, fmt.Errorf("Unable to read response body: %w", err)
    }

    var status CurrentLiveResponse
    if err := json.Unmarshal(body, &status); err != nil {
        return channel, nil, fmt.Errorf("Unable to parse response body: %w", err)
    }

    return channel, &status, nil
}

func (c *Crawler) isNew(channel *Channel, resp *CurrentLiveResponse) bool {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.state.IsNewStream(channel.Name, resp.Movie.ID) {
        log.Printf("Still live (ID: %s)", resp.Movie.ID)
        return false
    }
    log.Printf("New live (ID: %s)", resp.Movie.ID)

    if err := c.state.SetLatest(channel.Name, resp.Movie.ID); err != nil {
        log.Printf("Failed to update latest stream: %v", err)
        return false
    }
    return true
}

func (c *Crawler) crawlLoop(app Credentials) {
    delay := app.Delay.Duration
    if delay < time.Second {
        log.Printf("App %s has less than 1s of delay, increasing to 1s", app.ClientID)
        delay = time.Second
    }
    for {
        time.Sleep(app.Delay.Duration)

        channel, resp, err := c.fetchNext(&app)
        if err != nil {
            log.Printf("Failed: %v", err)
            continue
        }
        if resp == nil {
            log.Println("Not live")
            continue
        }

        if !c.isNew(channel, resp) {
            continue
        }

        if err := c.record(channel, resp); err != nil {
            log.Printf("Failed to start recording: %v", err)
            c.state.SetLatest(channel.Name, "<latest download failed to start, dummy value to trigger retry>")
        }
    }
}

func (c *Crawler) Crawl() {
    for _, app := range c.cfg.Apps {
        go c.crawlLoop(app)
    }
}

