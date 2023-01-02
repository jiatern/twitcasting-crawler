package main

import (
    "flag"
    "log"
)

func main() {
    configPath := flag.String("config", "config.json", "Path to configuration file")
    statePath := flag.String("state", "state.json", "Path to state file")
    flag.Parse()


    cfg, err := LoadConfig(*configPath)
    if err != nil {
        panic(err)
    }
    if len(cfg.Apps) == 0 {
        log.Println("ERROR: No apps configured")
        return
    }
    if len(cfg.Channels) == 0 {
        log.Println("ERROR: No channels configured")
    }

    state, err := LoadState(*statePath)
    if err != nil {
        panic(err)
    }

    crawler := &Crawler {
        cfg:   cfg,
        state: state,
    }
    crawler.Crawl()

    select{}
}
