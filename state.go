package main

import (
    "encoding/json"
    "os"
)

type State struct {
    path         string
    LastStream map[string]string `json:"last_stream"`
}

func LoadState(path string) (*State, error) {
    s := &State {
        path:       path,
        LastStream: make(map[string]string),
    }

    _, err := os.Stat(path)
    if os.IsNotExist(err) {
        return s, nil
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    if err := json.Unmarshal(data, s); err != nil {
        return nil, err
    }
    return s, nil
}

func (s *State) IsNewStream(channel string, id string) bool {
    return s.LastStream[channel] != id
}

func (s *State) SetLatest(channel string, id string) error {
    s.LastStream[channel] = id
    data, err := json.Marshal(s)
    if err != nil {
        return err
    }
    return os.WriteFile(s.path, data, 0644)
}

