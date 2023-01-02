# Usage

- Compile (`go build`) or download it (https://github.com/notpeko/twitcasting-crawler/releases)
- Create an [application](https://en.twitcasting.tv/developer.php)
- Fill config.json (see [config.json.example](https://github.com/notpeko/twitcasting-crawler/blob/master/config.json.example) for documentation)
- Install [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- Run it

# Minimal config

```json
{
    "apps": [
        {
            "client_id": "...",
            "client_secret": "..."
        }
    ],
    "channels": [
        "kuroneko_datenn",
        "whatweneed8837"
    ]
}
```

